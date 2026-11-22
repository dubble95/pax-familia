package com.paxfamilia.agent

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build

class PaxApp : Application() {

    companion object {
        const val CHANNEL_ID = "pax_service_channel"
        const val CHANNEL_NAME = "PaxFamilia 통합 스크린타임 서비스"
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                CHANNEL_NAME,
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "자녀의 스크린타임 잔여 시간을 실시간 동기화하고 기기 상태를 보호합니다."
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager?.createNotificationChannel(channel)
        }
    }
}
