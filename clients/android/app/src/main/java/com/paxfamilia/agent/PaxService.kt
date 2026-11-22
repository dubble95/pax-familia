package com.paxfamilia.agent

import android.app.Notification
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.gson.Gson
import com.paxfamilia.agent.model.DeviceType
import com.paxfamilia.agent.model.HeartbeatRequest
import com.paxfamilia.agent.model.HeartbeatResponse
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

class PaxService : Service() {

    companion object {
        private const val TAG = "PaxService"
        private const val NOTIFICATION_ID = 1001

        const val ACTION_START = "com.paxfamilia.agent.ACTION_START"
        const val ACTION_STOP = "com.paxfamilia.agent.ACTION_STOP"
    }

    private val serviceJob = Job()
    private val scope = CoroutineScope(Dispatchers.IO + serviceJob)

    private val gson = Gson()
    private val httpClient = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(5, TimeUnit.SECONDS)
        .build()

    private lateinit var usageTracker: UsageTracker

    private var isScreenOn = true
    private var isCurrentlyLocked = false
    private var lastTickTime = System.currentTimeMillis()

    // 화면 켜짐/꺼짐 수신 리시버 (배터리 절약 및 정확한 사용시간 측정)
    private val screenReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            when (intent?.action) {
                Intent.ACTION_SCREEN_ON -> {
                    isScreenOn = true
                    lastTickTime = System.currentTimeMillis()
                    Log.d(TAG, "📱 화면 켜짐: 시간 측정 재개")
                }
                Intent.ACTION_SCREEN_OFF -> {
                    isScreenOn = false
                    Log.d(TAG, "💤 화면 꺼짐: 시간 측정 일시 정지 (Doze 모드)")
                }
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        usageTracker = UsageTracker(this)

        val screenFilter = IntentFilter().apply {
            addAction(Intent.ACTION_SCREEN_ON)
            addAction(Intent.ACTION_SCREEN_OFF)
        }
        registerReceiver(screenReceiver, screenFilter)

        startForeground(NOTIFICATION_ID, buildNotification("PaxFamilia 활성화됨: 스크린타임 보호 중"))
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopSelf()
            return START_NOT_STICKY
        }

        startHeartbeatLoop()
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        serviceJob.cancel()
        try {
            unregisterReceiver(screenReceiver)
        } catch (e: Exception) {
            // Already unregistered
        }
        Log.d(TAG, "PaxService 종료됨")
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private var currentIntervalMs = 15000L

    private fun startHeartbeatLoop() {
        scope.launch {
            while (isActive) {
                try {
                    val nextSec = tick()
                    if (nextSec > 0) {
                        currentIntervalMs = nextSec * 1000L
                    }
                } catch (e: Exception) {
                    Log.e(TAG, "하트비트 루프 오류: ${e.message}")
                }
                delay(currentIntervalMs)
            }
        }
    }

    private fun tick(): Int {
        val prefs = getSharedPreferences("pax_prefs", Context.MODE_PRIVATE)
        val serverUrl = prefs.getString("server_url", "http://10.0.2.2:8080") ?: "http://10.0.2.2:8080"
        val childId = prefs.getString("child_id", "child_1") ?: "child_1"
        val deviceId = prefs.getString("device_id", "dev_android_1") ?: "dev_android_1"
        val deviceName = prefs.getString("device_name", "안드로이드 기기") ?: "안드로이드 기기"
        val deviceType = prefs.getString("device_type", DeviceType.ANDROID_PHONE) ?: DeviceType.ANDROID_PHONE

        val now = System.currentTimeMillis()
        var elapsedSec = ((now - lastTickTime) / 1000).toInt()
        lastTickTime = now

        if (elapsedSec > 60) elapsedSec = 60
        if (!isScreenOn || isCurrentlyLocked) {
            elapsedSec = 0
        }

        val currentApp = if (isScreenOn) usageTracker.getForegroundAppName() else "(화면 꺼짐)"
        val batteryPct = getBatteryLevel()

        val req = HeartbeatRequest(
            deviceId = deviceId,
            childId = childId,
            deviceName = deviceName,
            deviceType = deviceType,
            activeSeconds = elapsedSec,
            currentApp = currentApp,
            batteryLevel = batteryPct,
            isIdle = !isScreenOn
        )

        try {
            val jsonBody = gson.toJson(req)
            val request = Request.Builder()
                .url("${serverUrl.trimEnd('/')}/api/heartbeat")
                .post(jsonBody.toRequestBody("application/json".toMediaType()))
                .build()

            val response = httpClient.newCall(request).execute()
            if (response.isSuccessful) {
                val respBody = response.body?.string()
                val resp = gson.fromJson(respBody, HeartbeatResponse::class.java)
                handleServerResponse(resp)
                return resp.nextCheckSeconds ?: 15
            } else {
                Log.w(TAG, "서버 응답 비정상: ${response.code}")
            }
        } catch (e: Exception) {
            Log.w(TAG, "서버 연결 실패 (오프라인 모드): ${e.message}")
        }
        return 15
    }

    private fun handleServerResponse(resp: HeartbeatResponse) {
        val remainingMin = resp.remainingMinutes
        updateNotification("오늘 남은 시간: ${remainingMin}분")

        if (resp.shouldLock) {
            if (!isCurrentlyLocked) {
                isCurrentlyLocked = true
                showLockOverlay(resp.lockReason ?: "시간이 소진되었습니다.")
            }
        } else {
            if (isCurrentlyLocked) {
                isCurrentlyLocked = false
                dismissLockOverlay()
            }
        }
    }

    private fun showLockOverlay(reason: String) {
        val intent = Intent(this, OverlayLockActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra(OverlayLockActivity.EXTRA_LOCK_REASON, reason)
        }
        startActivity(intent)
    }

    private fun dismissLockOverlay() {
        val intent = Intent(OverlayLockActivity.ACTION_UNLOCK)
        sendBroadcast(intent)
    }

    private fun getBatteryLevel(): Int {
        val bm = getSystemService(Context.BATTERY_SERVICE) as? BatteryManager
        return bm?.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY) ?: -1
    }

    private fun buildNotification(text: String): Notification {
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, PaxApp.CHANNEL_ID)
            .setContentTitle("🕊️ PaxFamilia 스크린타임")
            .setContentText(text)
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .build()
    }

    private fun updateNotification(text: String) {
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as? android.app.NotificationManager
        manager?.notify(NOTIFICATION_ID, buildNotification(text))
    }
}
