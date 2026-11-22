package com.paxfamilia.agent.model

import com.google.gson.annotations.SerializedName

object DeviceType {
    const val ANDROID_PHONE = "android_phone"
    const val ANDROID_TABLET = "android_tablet"
    const val WINDOWS_PC = "windows_pc"
}

object DeviceStatus {
    const val ONLINE = "online"
    const val IDLE = "idle"
    const val LOCKED = "locked"
    const val OFFLINE = "offline"
}

data class HeartbeatRequest(
    @SerializedName("device_id") val deviceId: String,
    @SerializedName("child_id") val childId: String,
    @SerializedName("device_name") val deviceName: String,
    @SerializedName("device_type") val deviceType: String,
    @SerializedName("active_seconds") val activeSeconds: Int,
    @SerializedName("current_app") val currentApp: String,
    @SerializedName("battery_level") val batteryLevel: Int,
    @SerializedName("is_idle") val isIdle: Boolean
)

data class HeartbeatResponse(
    @SerializedName("remaining_minutes") val remainingMinutes: Int,
    @SerializedName("remaining_seconds") val remainingSeconds: Int,
    @SerializedName("should_lock") val shouldLock: Boolean,
    @SerializedName("lock_reason") val lockReason: String?,
    @SerializedName("warning_message") val warningMessage: String?,
    @SerializedName("next_check_seconds") val nextCheckSeconds: Int?
)

data class Child(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("daily_quota_minutes") val dailyQuotaMinutes: Int,
    @SerializedName("bonus_minutes") val bonusMinutes: Int,
    @SerializedName("used_seconds") val usedSeconds: Int,
    @SerializedName("force_locked") val forceLocked: Boolean,
    @SerializedName("last_reset_date") val lastResetDate: String,
    @SerializedName("updated_at") val updatedAt: String
)

data class Device(
    @SerializedName("id") val id: String,
    @SerializedName("child_id") val childId: String,
    @SerializedName("name") val name: String,
    @SerializedName("type") val type: String,
    @SerializedName("status") val status: String,
    @SerializedName("current_app") val currentApp: String,
    @SerializedName("battery_level") val batteryLevel: Int,
    @SerializedName("last_seen") val lastSeen: String
)

data class SystemStatus(
    @SerializedName("child") val child: Child?,
    @SerializedName("devices") val devices: List<Device>?
)
