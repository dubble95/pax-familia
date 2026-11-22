package com.paxfamilia.agent

import android.app.AppOpsManager
import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Process

class UsageTracker(private val context: Context) {

    private val usageStatsManager =
        context.getSystemService(Context.USAGE_STATS_SERVICE) as? UsageStatsManager

    // 자주 사용되는 주요 앱의 친근한 한글 이름 매핑
    private val appNameMap = mapOf(
        "com.google.android.youtube" to "YouTube",
        "com.google.android.apps.youtube.kids" to "YouTube Kids",
        "com.roblox.client" to "Roblox (로블록스)",
        "com.supercell.brawlstars" to "브롤스타즈",
        "com.mojang.minecraftpe" to "마인크래프트",
        "com.kakao.talk" to "카카오톡",
        "com.instagram.android" to "인스타그램",
        "com.ss.android.ugc.trill" to "틱톡",
        "com.netflix.mediaclient" to "넷플릭스",
        "com.android.chrome" to "크롬 브라우저",
        "com.sec.android.app.sbrowser" to "삼성 인터넷",
        "com.nhn.android.webtoon" to "네이버 웹툰"
    )

    /**
     * 사용 정보 접근 권한(PACKAGE_USAGE_STATS)이 허용되어 있는지 확인
     */
    fun hasUsageStatsPermission(): Boolean {
        val appOps = context.getSystemService(Context.APP_OPS_SERVICE) as? AppOpsManager ?: return false
        val mode = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            appOps.unsafeCheckOpNoThrow(
                AppOpsManager.OPSTR_GET_USAGE_STATS,
                Process.myUid(),
                context.packageName
            )
        } else {
            appOps.checkOpNoThrow(
                AppOpsManager.OPSTR_GET_USAGE_STATS,
                Process.myUid(),
                context.packageName
            )
        }
        return mode == AppOpsManager.MODE_ALLOWED
    }

    /**
     * 현재 포그라운드에서 실행 중인 앱의 이름 반환
     */
    fun getForegroundAppName(): String {
        if (!hasUsageStatsPermission() || usageStatsManager == null) {
            return "권한 미부여"
        }

        val endTime = System.currentTimeMillis()
        val startTime = endTime - 1000 * 30 // 최근 30초 내 이벤트

        val events = usageStatsManager.queryEvents(startTime, endTime)
        val event = UsageEvents.Event()
        var lastForegroundPackage: String? = null

        while (events.hasNextEvent()) {
            events.getNextEvent(event)
            if (event.eventType == UsageEvents.Event.ACTIVITY_RESUMED) {
                lastForegroundPackage = event.packageName
            }
        }

        if (lastForegroundPackage == null) {
            return "바탕화면 / 대기 중"
        }

        // 자체 잠금 화면이나 기본 런처인 경우
        if (lastForegroundPackage == context.packageName) {
            return "PaxFamilia 잠금 화면"
        }

        // 사전 정의된 한글 앱 이름 확인
        appNameMap[lastForegroundPackage]?.let { return it }

        // PackageManager를 통해 앱 라벨 조회
        return try {
            val pm = context.packageManager
            val appInfo = pm.getApplicationInfo(lastForegroundPackage, 0)
            pm.getApplicationLabel(appInfo).toString()
        } catch (e: PackageManager.NameNotFoundException) {
            lastForegroundPackage
        }
    }
}
