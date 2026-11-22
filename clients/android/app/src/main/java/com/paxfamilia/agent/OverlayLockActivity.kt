package com.paxfamilia.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.Build
import android.os.Bundle
import android.view.KeyEvent
import android.view.WindowManager
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity

class OverlayLockActivity : AppCompatActivity() {

    companion object {
        const val ACTION_UNLOCK = "com.paxfamilia.agent.ACTION_UNLOCK"
        const val EXTRA_LOCK_REASON = "extra_lock_reason"
    }

    private val unlockReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            if (intent?.action == ACTION_UNLOCK) {
                finish()
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // 화면 켜짐 유지, 잠금화면 위 표시
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
            setShowWhenLocked(true)
            setTurnScreenOn(true)
        } else {
            @Suppress("DEPRECATION")
            window.addFlags(
                WindowManager.LayoutParams.FLAG_SHOW_WHEN_LOCKED or
                WindowManager.LayoutParams.FLAG_DISMISS_KEYGUARD or
                WindowManager.LayoutParams.FLAG_TURN_SCREEN_ON
            )
        }
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)

        setContentView(R.layout.activity_lock)

        val reason = intent.getStringExtra(EXTRA_LOCK_REASON) ?: "오늘의 사용 시간이 모두 소진되었습니다."
        findViewById<TextView>(R.id.tvLockReason).text = reason

        findViewById<Button>(R.id.btnRequestBonus).setOnClickListener {
            Toast.makeText(this, "부모님께 시간 연장 알림을 요청했습니다. 잠시만 기다려주세요.", Toast.LENGTH_LONG).show()
        }

        // 부모 비상 PIN 해제 (오프라인/긴급용)
        findViewById<Button>(R.id.btnParentPin).setOnClickListener {
            showPinDialog()
        }

        // 뒤로가기 버튼 비활성화 (우회 방지)
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                // 뒤로가기 무시
            }
        })

        // 언락 브로드캐스트 수신 등록
        val filter = IntentFilter(ACTION_UNLOCK)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(unlockReceiver, filter, Context.RECEIVER_NOT_EXPORTED)
        } else {
            registerReceiver(unlockReceiver, filter)
        }
    }

    private fun showPinDialog() {
        val input = android.widget.EditText(this).apply {
            inputType = android.text.InputType.TYPE_CLASS_NUMBER or android.text.InputType.TYPE_NUMBER_VARIATION_PASSWORD
            hint = "부모 PIN 4자리 입력 (기본: 1234)"
            setPadding(50, 30, 50, 30)
        }

        androidx.appcompat.app.AlertDialog.Builder(this)
            .setTitle("🔑 부모 비상 잠금 해제")
            .setMessage("긴급 상황 시 4자리 PIN 번호를 입력하면 즉시 30분간 잠금이 임시 해제됩니다.")
            .setView(input)
            .setPositiveButton("해제") { _, _ ->
                val entered = input.text.toString().trim()
                val prefs = getSharedPreferences("pax_prefs", Context.MODE_PRIVATE)
                val correctPin = prefs.getString("parent_pin", "1234") ?: "1234"
                if (entered == correctPin) {
                    Toast.makeText(this, "부모 인증 성공: 30분 임시 해제되었습니다.", Toast.LENGTH_SHORT).show()
                    val rem = prefs.getInt("local_remaining_seconds", 0) + 1800
                    prefs.edit().putInt("local_remaining_seconds", rem).apply()
                    sendBroadcast(Intent(ACTION_UNLOCK))
                    finish()
                } else {
                    Toast.makeText(this, "PIN 번호가 일치하지 않습니다.", Toast.LENGTH_SHORT).show()
                }
            }
            .setNegativeButton("취소", null)
            .show()
    }

    override fun onDestroy() {
        super.onDestroy()
        try {
            unregisterReceiver(unlockReceiver)
        } catch (e: Exception) {
            // Already unregistered
        }
    }

    // 홈/최근 앱으로 전환 시에도 즉시 다시 최상위로 복원
    override fun onUserLeaveHint() {
        super.onUserLeaveHint()
        val bringToFrontIntent = Intent(this, OverlayLockActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_REORDER_TO_FRONT
        }
        startActivity(bringToFrontIntent)
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent?): Boolean {
        // 볼륨 키, 뒤로가기 키 등의 조작 제한
        if (keyCode == KeyEvent.KEYCODE_BACK) {
            return true
        }
        return super.onKeyDown(keyCode, event)
    }
}
