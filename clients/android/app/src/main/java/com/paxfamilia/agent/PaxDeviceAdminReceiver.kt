package com.paxfamilia.agent

import android.app.admin.DeviceAdminReceiver
import android.content.Context
import android.content.Intent
import android.widget.Toast

class PaxDeviceAdminReceiver : DeviceAdminReceiver() {

    override fun onEnabled(context: Context, intent: Intent) {
        super.onEnabled(context, intent)
        Toast.makeText(context, "PaxFamilia 기기 관리자 보호가 활성화되었습니다.", Toast.LENGTH_SHORT).show()
    }

    override fun onDisableRequested(context: Context, intent: Intent): CharSequence {
        return "PaxFamilia 기기 관리자 권한을 비활성화하면 자녀 기기 보호 및 스크린타임 관리가 중단됩니다. 부모님의 동의가 필요합니다."
    }

    override fun onDisabled(context: Context, intent: Intent) {
        super.onDisabled(context, intent)
        Toast.makeText(context, "PaxFamilia 기기 관리자 보호가 해제되었습니다.", Toast.LENGTH_SHORT).show()
    }
}
