package com.paxfamilia.agent

import android.app.admin.DevicePolicyManager
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.PowerManager
import android.provider.Settings
import android.widget.Button
import android.widget.EditText
import android.widget.RadioGroup
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.paxfamilia.agent.model.DeviceType

class MainActivity : AppCompatActivity() {

    private lateinit var etServerUrl: EditText
    private lateinit var etChildId: EditText
    private lateinit var etDeviceName: EditText
    private lateinit var rgDeviceType: RadioGroup

    private lateinit var tvUsagePerm: TextView
    private lateinit var tvOverlayPerm: TextView
    private lateinit var tvBatteryPerm: TextView
    private lateinit var tvAdminPerm: TextView

    private lateinit var btnStartService: Button

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        initViews()
        loadPreferences()
        setupListeners()
    }

    override fun onResume() {
        super.onResume()
        updatePermissionStatuses()
    }

    private fun initViews() {
        etServerUrl = findViewById(R.id.etServerUrl)
        etChildId = findViewById(R.id.etChildId)
        etDeviceName = findViewById(R.id.etDeviceName)
        rgDeviceType = findViewById(R.id.rgDeviceType)

        tvUsagePerm = findViewById(R.id.tvUsagePerm)
        tvOverlayPerm = findViewById(R.id.tvOverlayPerm)
        tvBatteryPerm = findViewById(R.id.tvBatteryPerm)
        tvAdminPerm = findViewById(R.id.tvAdminPerm)

        btnStartService = findViewById(R.id.btnStartService)
    }

    private fun loadPreferences() {
        val prefs = getSharedPreferences("pax_prefs", Context.MODE_PRIVATE)
        etServerUrl.setText(prefs.getString("server_url", "https://pax-familia.your-subdomain.workers.dev"))
        etChildId.setText(prefs.getString("child_id", "child_1"))
        etDeviceName.setText(prefs.getString("device_name", Build.MODEL ?: "자녀 스마트폰"))

        val type = prefs.getString("device_type", DeviceType.ANDROID_PHONE)
        if (type == DeviceType.ANDROID_TABLET) {
            rgDeviceType.check(R.id.rbTablet)
        } else {
            rgDeviceType.check(R.id.rbPhone)
        }
    }

    private fun savePreferences() {
        val prefs = getSharedPreferences("pax_prefs", Context.MODE_PRIVATE)
        val devType = if (rgDeviceType.checkedRadioButtonId == R.id.rbTablet) {
            DeviceType.ANDROID_TABLET
        } else {
            DeviceType.ANDROID_PHONE
        }

        prefs.edit()
            .putString("server_url", etServerUrl.text.toString().trim())
            .putString("child_id", etChildId.text.toString().trim())
            .putString("device_name", etDeviceName.text.toString().trim())
            .putString("device_type", devType)
            .apply()
    }

    private fun setupListeners() {
        findViewById<Button>(R.id.btnReqUsage).setOnClickListener {
            startActivity(Intent(Settings.ACTION_USAGE_ACCESS_SETTINGS))
        }

        findViewById<Button>(R.id.btnReqOverlay).setOnClickListener {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                val intent = Intent(
                    Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
                    Uri.parse("package:$packageName")
                )
                startActivity(intent)
            }
        }

        findViewById<Button>(R.id.btnReqBattery).setOnClickListener {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                val intent = Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS).apply {
                    data = Uri.parse("package:$packageName")
                }
                startActivity(intent)
            }
        }

        findViewById<Button>(R.id.btnReqAdmin).setOnClickListener {
            val componentName = ComponentName(this, PaxDeviceAdminReceiver::class.java)
            val intent = Intent(DevicePolicyManager.ACTION_ADD_DEVICE_ADMIN).apply {
                putExtra(DevicePolicyManager.EXTRA_DEVICE_ADMIN, componentName)
                putExtra(DevicePolicyManager.EXTRA_ADD_EXPLANATION, "PaxFamilia 자녀 보호 에이전트의 임의 삭제를 방지합니다.")
            }
            startActivity(intent)
        }

        btnStartService.setOnClickListener {
            savePreferences()
            val serviceIntent = Intent(this, PaxService::class.java).apply {
                action = PaxService.ACTION_START
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(serviceIntent)
            } else {
                startService(serviceIntent)
            }
            Toast.makeText(this, "🕊️ PaxFamilia 보호 서비스가 시작되었습니다!", Toast.LENGTH_SHORT).show()
        }
    }

    private fun updatePermissionStatuses() {
        // 1. Usage stats
        val tracker = UsageTracker(this)
        val hasUsage = tracker.hasUsageStatsPermission()
        tvUsagePerm.text = if (hasUsage) "✅ 허용됨" else "❌ 권한 필요"

        // 2. Overlay
        val hasOverlay = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            Settings.canDrawOverlays(this)
        } else true
        tvOverlayPerm.text = if (hasOverlay) "✅ 허용됨" else "❌ 권한 필요"

        // 3. Battery optimization
        val pm = getSystemService(Context.POWER_SERVICE) as? PowerManager
        val hasBattery = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            pm?.isIgnoringBatteryOptimizations(packageName) == true
        } else true
        tvBatteryPerm.text = if (hasBattery) "✅ 최적화 제외됨" else "⚠️ 최적화 해제 권장"

        // 4. Device Admin
        val dpm = getSystemService(Context.DEVICE_POLICY_SERVICE) as? DevicePolicyManager
        val adminComp = ComponentName(this, PaxDeviceAdminReceiver::class.java)
        val hasAdmin = dpm?.isAdminActive(adminComp) == true
        tvAdminPerm.text = if (hasAdmin) "✅ 등록됨" else "⚠️ 등록 권장 (삭제 방지)"
    }
}
