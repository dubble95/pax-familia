/**
 * 🕊️ PaxFamilia - Cloudflare Workers Serverless Backend
 * 
 * 크로스 플랫폼 스크린타임 통합 관리 서버리스 백엔드
 * - Cloudflare Workers KV 기반 상태 영속성 지원 (바인딩 없을 시 인메모리 폴백)
 * - 내장 부모 대시보드 웹 (GET /)
 * - 텔레그램 봇 Webhook 지원 (POST /api/telegram) - 원클릭 잠금 및 보너스 제어
 * - 100% 무료 티어로 평생 운영 가능 (홈서버 / 라즈베리파이 불필요)
 */

// 기본 초기 상태 데이터
const DEFAULT_STATE = {
  children: {
    child_1: {
      id: "child_1",
      name: "우리 아이",
      daily_quota_minutes: 120,
      bonus_minutes: 0,
      used_seconds: 0,
      force_locked: false,
      last_reset_date: getTodayString(),
      updated_at: new Date().toISOString()
    }
  },
  devices: {
    dev_phone_1: {
      id: "dev_phone_1",
      child_id: "child_1",
      name: "안드로이드 스마트폰",
      type: "android_phone",
      status: "offline",
      current_app: "대기 중",
      battery_level: -1,
      last_seen: new Date(Date.now() - 3600000).toISOString()
    },
    dev_tablet_1: {
      id: "dev_tablet_1",
      child_id: "child_1",
      name: "안드로이드 태블릿",
      type: "android_tablet",
      status: "offline",
      current_app: "대기 중",
      battery_level: -1,
      last_seen: new Date(Date.now() - 3600000).toISOString()
    },
    dev_pc_1: {
      id: "dev_pc_1",
      child_id: "child_1",
      name: "윈도우 PC",
      type: "windows_pc",
      status: "offline",
      current_app: "대기 중",
      battery_level: -1,
      last_seen: new Date(Date.now() - 3600000).toISOString()
    }
  }
};

// 인메모리 폴백 (KV 미설정 환경)
let memoryState = JSON.parse(JSON.stringify(DEFAULT_STATE));

function getTodayString() {
  try {
    const formatter = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Seoul",
      year: "numeric",
      month: "2-digit",
      day: "2-digit"
    });
    return formatter.format(new Date());
  } catch (e) {
    return new Date().toISOString().slice(0, 10);
  }
}

async function loadState(env) {
  if (env && env.PAX_KV) {
    const kvData = await env.PAX_KV.get("pax_state", { type: "json" });
    if (kvData) {
      return kvData;
    }
  }
  return memoryState;
}

async function saveState(env, state) {
  if (env && env.PAX_KV) {
    await env.PAX_KV.put("pax_state", JSON.stringify(state));
  }
  memoryState = state;
}

function checkDailyReset(child) {
  const today = getTodayString();
  if (child.last_reset_date !== today) {
    child.last_reset_date = today;
    child.used_seconds = 0;
    child.bonus_minutes = 0;
    child.force_locked = false;
    child.updated_at = new Date().toISOString();
  }
}

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type, Authorization"
};

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      ...CORS_HEADERS
    }
  });
}

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // OPTIONS preflight
    if (request.method === "OPTIONS") {
      return new Response(null, { headers: CORS_HEADERS });
    }

    // 1. 내장 웹 대시보드 서빙 (GET /)
    if (url.pathname === "/" || url.pathname === "/index.html") {
      return new Response(DASHBOARD_HTML, {
        headers: {
          "Content-Type": "text/html; charset=utf-8",
          ...CORS_HEADERS
        }
      });
    }

    // 2. Heartbeat API (POST /api/heartbeat)
    if (url.pathname === "/api/heartbeat" && request.method === "POST") {
      let body;
      try {
        body = await request.json();
      } catch (e) {
        return jsonResponse({ error: "Invalid JSON" }, 400);
      }

      const state = await loadState(env);
      const childID = body.child_id || "child_1";

      if (!state.children[childID]) {
        state.children[childID] = {
          id: childID,
          name: "우리 아이",
          daily_quota_minutes: 120,
          bonus_minutes: 0,
          used_seconds: 0,
          force_locked: false,
          last_reset_date: getTodayString(),
          updated_at: new Date().toISOString()
        };
      }

      const child = state.children[childID];
      checkDailyReset(child);

      // 유휴(Idle)가 아니고 활성 초가 있는 경우 시간 차감
      if (!body.is_idle && body.active_seconds > 0) {
        child.used_seconds += body.active_seconds;
        child.updated_at = new Date().toISOString();
      }

      // 기기 정보 등록 및 업데이트
      const devID = body.device_id || "unknown_device";
      if (!state.devices[devID]) {
        state.devices[devID] = {
          id: devID,
          child_id: childID,
          name: body.device_name || body.device_type || "기기",
          type: body.device_type || "windows_pc",
          status: "offline",
          current_app: "대기 중",
          battery_level: -1,
          last_seen: new Date().toISOString()
        };
      }

      const dev = state.devices[devID];
      dev.last_seen = new Date().toISOString();
      if (body.current_app) dev.current_app = body.current_app;
      if (typeof body.battery_level === "number") dev.battery_level = body.battery_level;

      // 잔여 시간 계산
      const totalAllowedMinutes = child.daily_quota_minutes + child.bonus_minutes;
      const totalAllowedSeconds = totalAllowedMinutes * 60;
      let remainingSeconds = totalAllowedSeconds - child.used_seconds;
      if (remainingSeconds < 0) remainingSeconds = 0;
      const remainingMinutes = Math.ceil(remainingSeconds / 60);

      // 잠금 여부 판정
      let shouldLock = false;
      let lockReason = "";
      let warningMsg = "";

      if (child.force_locked) {
        shouldLock = true;
        lockReason = "부모에 의한 즉시 잠금";
        dev.status = "locked";
      } else if (remainingSeconds <= 0) {
        shouldLock = true;
        lockReason = "일일 허용 사용시간 모두 소진";
        dev.status = "locked";
      } else {
        dev.status = body.is_idle ? "idle" : "online";
        if (remainingMinutes <= 5) {
          warningMsg = `남은 시간이 ${remainingMinutes}분 남았습니다. 작업을 마무리하세요.`;
        }
      }

      await saveState(env, state);

      return jsonResponse({
        remaining_minutes: remainingMinutes,
        remaining_seconds: remainingSeconds,
        should_lock: shouldLock,
        lock_reason: lockReason,
        warning_message: warningMsg,
        next_check_seconds: 15
      });
    }

    // 3. Status API (GET /api/status)
    if (url.pathname === "/api/status" && request.method === "GET") {
      const state = await loadState(env);
      const childID = url.searchParams.get("child_id") || "child_1";

      if (!state.children[childID]) {
        state.children[childID] = {
          id: childID,
          name: "우리 아이",
          daily_quota_minutes: 120,
          bonus_minutes: 0,
          used_seconds: 0,
          force_locked: false,
          last_reset_date: getTodayString(),
          updated_at: new Date().toISOString()
        };
      }

      const child = state.children[childID];
      checkDailyReset(child);

      const now = Date.now();
      const devList = Object.values(state.devices)
        .filter(d => d.child_id === childID)
        .map(d => {
          const copy = { ...d };
          // 60초 이상 미접속 시 오프라인 처리
          const diff = now - new Date(copy.last_seen).getTime();
          if (diff > 60000) {
            copy.status = "offline";
          }
          return copy;
        });

      // ⭐️ 기기 순서 고정 정렬 (대시보드 순서 뒤바뀜 방지)
      const typeRank = { 'android_phone': 1, 'android_tablet': 2, 'windows_pc': 3 };
      devList.sort((a, b) => {
        if (a.created_at && b.created_at && a.created_at !== b.created_at) {
          return new Date(a.created_at) - new Date(b.created_at);
        }
        const r1 = typeRank[a.type] || 4;
        const r2 = typeRank[b.type] || 4;
        if (r1 !== r2) return r1 - r2;
        return (a.id || '').localeCompare(b.id || '');
      });

      return jsonResponse({
        child,
        devices: devList
      });
    }

    // 4. Force Lock 토글 (POST /api/lock)
    if (url.pathname === "/api/lock" && request.method === "POST") {
      const body = await request.json();
      const childID = body.child_id || "child_1";
      const state = await loadState(env);
      if (state.children[childID]) {
        state.children[childID].force_locked = !!body.lock;
        state.children[childID].updated_at = new Date().toISOString();
        await saveState(env, state);
      }
      return jsonResponse({ success: true, force_locked: body.lock });
    }

    // 5. Bonus 시간 부여 (POST /api/bonus)
    if (url.pathname === "/api/bonus" && request.method === "POST") {
      const body = await request.json();
      const childID = body.child_id || "child_1";
      const minutes = parseInt(body.minutes, 10) || 0;
      const state = await loadState(env);
      if (state.children[childID]) {
        state.children[childID].bonus_minutes += minutes;
        state.children[childID].updated_at = new Date().toISOString();
        await saveState(env, state);
      }
      return jsonResponse({ success: true, bonus_minutes: state.children[childID]?.bonus_minutes });
    }

    // 6. Quota 일일 할당 변경 (POST /api/quota)
    if (url.pathname === "/api/quota" && request.method === "POST") {
      const body = await request.json();
      const childID = body.child_id || "child_1";
      const minutes = parseInt(body.minutes, 10) || 120;
      const state = await loadState(env);
      if (state.children[childID]) {
        state.children[childID].daily_quota_minutes = minutes;
        state.children[childID].updated_at = new Date().toISOString();
        await saveState(env, state);
      }
      return jsonResponse({ success: true, daily_quota_minutes: minutes });
    }

    // 7. Reset 오늘 사용량 초기화 (GET / POST /api/reset)
    if (url.pathname === "/api/reset") {
      const childID = url.searchParams.get("child_id") || "child_1";
      const state = await loadState(env);
      if (state.children[childID]) {
        state.children[childID].used_seconds = 0;
        state.children[childID].bonus_minutes = 0;
        state.children[childID].force_locked = false;
        state.children[childID].updated_at = new Date().toISOString();
        await saveState(env, state);
      }
      return jsonResponse({ success: true });
    }

    // 8. 텔레그램 봇 Webhook 핸들러 (POST /api/telegram)
    if (url.pathname === "/api/telegram" && request.method === "POST") {
      try {
        const update = await request.json();
        const tgToken = env.TELEGRAM_BOT_TOKEN;
        if (!tgToken) {
          return jsonResponse({ ok: false, error: "TELEGRAM_BOT_TOKEN not configured" });
        }

        await handleTelegramUpdate(update, tgToken, env);
        return jsonResponse({ ok: true });
      } catch (err) {
        return jsonResponse({ ok: false, error: err.message }, 500);
      }
    }

    return new Response("Not Found", { status: 404 });
  }
};

/**
 * 텔레그램 Webhook 메시지 및 인라인 버튼 처리
 */
async function handleTelegramUpdate(update, token, env) {
  let chatId = null;
  let text = "";
  let callbackQueryId = null;

  if (update.message) {
    chatId = update.message.chat.id;
    text = (update.message.text || "").trim();
  } else if (update.callback_query) {
    chatId = update.callback_query.message.chat.id;
    text = update.callback_query.data;
    callbackQueryId = update.callback_query.id;
  }

  if (!chatId) return;

  const state = await loadState(env);
  const child = state.children["child_1"] || { daily_quota_minutes: 120, used_seconds: 0, bonus_minutes: 0 };
  checkDailyReset(child);

  const totalAllowed = (child.daily_quota_minutes + child.bonus_minutes) * 60;
  const remainingSeconds = Math.max(0, totalAllowed - child.used_seconds);
  const remainingMinutes = Math.ceil(remainingSeconds / 60);

  // 콜백 응답
  if (callbackQueryId) {
    await fetch(`https://api.telegram.org/bot${token}/answerCallbackQuery`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ callback_query_id: callbackQueryId, text: "명령 처리 완료!" })
    });
  }

  // 명령 분기
  if (text.startsWith("/lock") || text === "lock") {
    child.force_locked = true;
    child.updated_at = new Date().toISOString();
    await saveState(env, state);
    await sendTelegramMessage(token, chatId, "🛑 **모든 자녀 기기를 즉시 잠갔습니다.**", getTelegramKeyboard(true));
  } else if (text.startsWith("/unlock") || text === "unlock") {
    child.force_locked = false;
    child.updated_at = new Date().toISOString();
    await saveState(env, state);
    await sendTelegramMessage(token, chatId, `🟢 **잠금이 해제되었습니다.** (남은 시간: ${remainingMinutes}분)`, getTelegramKeyboard(false));
  } else if (text.startsWith("/bonus") || text.startsWith("bonus_")) {
    let addMins = 30;
    if (text.includes("15")) addMins = 15;
    if (text.includes("60")) addMins = 60;
    child.bonus_minutes += addMins;
    child.force_locked = false;
    child.updated_at = new Date().toISOString();
    await saveState(env, state);
    const newRemaining = Math.ceil(Math.max(0, (child.daily_quota_minutes + child.bonus_minutes) * 60 - child.used_seconds) / 60);
    await sendTelegramMessage(token, chatId, `🎁 **보너스 ${addMins}분이 지급되었습니다!**\n현재 남은 시간: **${newRemaining}분**`, getTelegramKeyboard(false));
  } else {
    // 기본 상태 조회
    const lockText = child.force_locked ? "🔴 즉시 잠김" : (remainingSeconds <= 0 ? "🔒 시간 소진으로 잠김" : "🟢 정상 이용 가능");
    const msg = `🕊️ **PaxFamilia 현재 현황**\n\n` +
      `👤 자녀: **${child.name}**\n` +
      `⏱️ 남은 시간: **${remainingMinutes}분** (기본 ${child.daily_quota_minutes}분 + 보너스 ${child.bonus_minutes}분)\n` +
      `⏳ 사용 시간: **${Math.floor(child.used_seconds / 60)}분**\n` +
      `🛡️ 상태: **${lockText}**\n\n` +
      `아래 버튼을 눌러 즉시 제어할 수 있습니다:`;
    await sendTelegramMessage(token, chatId, msg, getTelegramKeyboard(child.force_locked));
  }
}

function getTelegramKeyboard(isLocked) {
  return {
    inline_keyboard: [
      [
        isLocked
          ? { text: "🔓 잠금 해제", callback_data: "unlock" }
          : { text: "🛑 즉시 올스톱 잠금", callback_data: "lock" },
        { text: "🎁 +15분 보너스", callback_data: "bonus_15" },
        { text: "🎁 +30분 보너스", callback_data: "bonus_30" }
      ]
    ]
  };
}

async function sendTelegramMessage(token, chatId, text, replyMarkup) {
  await fetch(`https://api.telegram.org/bot${token}/sendMessage`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_id: chatId,
      text,
      parse_mode: "Markdown",
      reply_markup: replyMarkup
    })
  });
}

// 내장 웹 대시보드 HTML
const DASHBOARD_HTML = `<!DOCTYPE html>
<html lang="ko">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>PaxFamilia - 가정의 평화 대시보드 (Serverless)</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Pretendard:wght@400;600;700;800&display=swap');
    body { font-family: 'Pretendard', sans-serif; }
    .pulse-online { animation: pulse 2s infinite; }
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
  </style>
</head>
<body class="bg-slate-900 text-slate-100 min-h-screen">
  <header class="bg-slate-800/80 backdrop-blur border-b border-slate-700 sticky top-0 z-50">
    <div class="max-w-4xl mx-auto px-4 py-3 flex items-center justify-between">
      <div class="flex items-center space-x-3">
        <span class="text-2xl">🕊️</span>
        <div>
          <h1 class="text-lg font-bold tracking-tight text-white flex items-center gap-2">
            PaxFamilia
            <span class="text-xs bg-emerald-500/20 text-emerald-300 px-2 py-0.5 rounded-full font-medium border border-emerald-500/30">Serverless Cloud</span>
          </h1>
          <p class="text-xs text-slate-400">통합 크로스 플랫폼 스크린타임 관리</p>
        </div>
      </div>
      <div class="flex items-center space-x-2">
        <button onclick="fetchStatus()" class="p-2 text-slate-400 hover:text-white rounded-lg hover:bg-slate-700 transition" title="새로고침">
          <i class="fa-solid fa-arrows-rotate" id="refreshIcon"></i>
        </button>
      </div>
    </div>
  </header>

  <main class="max-w-4xl mx-auto px-4 py-6 space-y-6">
    <div class="bg-gradient-to-br from-slate-800 to-slate-850 rounded-2xl p-6 border border-slate-700 shadow-xl relative overflow-hidden">
      <div class="absolute -right-8 -top-8 w-40 h-40 bg-indigo-500/10 rounded-full blur-2xl pointer-events-none"></div>

      <div class="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <div class="flex items-center gap-2 mb-1">
            <span class="text-sm font-semibold text-indigo-400" id="childName">우리 아이</span>
            <span id="lockBadge" class="hidden text-xs bg-red-500/20 text-red-400 border border-red-500/30 px-2 py-0.5 rounded-full font-bold">
              <i class="fa-solid fa-lock mr-1"></i>잠김
            </span>
          </div>
          <div class="flex items-baseline gap-3">
            <span class="text-4xl md:text-5xl font-black tracking-tight" id="remainingTime">--분</span>
            <span class="text-sm text-slate-400 font-medium">오늘 남은 시간</span>
          </div>
          <p class="text-xs text-slate-400 mt-2 flex items-center gap-2">
            <span>기본 할당: <b id="dailyQuota">120분</b></span>
            <span>•</span>
            <span>보너스: <b id="bonusTime" class="text-emerald-400">+0분</b></span>
            <span>•</span>
            <span>사용한 시간: <b id="usedTime" class="text-amber-400">0분</b></span>
          </p>
        </div>

        <div class="flex flex-wrap gap-2">
          <button id="lockToggleBtn" onclick="toggleLock()" class="flex-1 md:flex-none px-4 py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition shadow-lg bg-red-600 hover:bg-red-500 text-white">
            <i class="fa-solid fa-hand"></i>
            <span id="lockBtnText">즉시 올스톱 잠금</span>
          </button>
          
          <div class="flex items-center gap-1 w-full md:w-auto">
            <button onclick="addBonus(15)" class="flex-1 md:flex-none px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-xl text-xs font-semibold text-emerald-300 transition">
              +15분
            </button>
            <button onclick="addBonus(30)" class="flex-1 md:flex-none px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-xl text-xs font-semibold text-emerald-300 transition">
              +30분
            </button>
            <button onclick="addBonus(60)" class="flex-1 md:flex-none px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-xl text-xs font-semibold text-emerald-300 transition">
              +1시간
            </button>
          </div>
        </div>
      </div>

      <div class="mt-6">
        <div class="flex justify-between text-xs text-slate-400 mb-1.5 font-medium">
          <span>진행률</span>
          <span id="progressText">0% 소진</span>
        </div>
        <div class="w-full bg-slate-700/60 rounded-full h-3 overflow-hidden p-0.5">
          <div id="progressBar" class="bg-indigo-500 h-full rounded-full transition-all duration-500" style="width: 0%"></div>
        </div>
      </div>
    </div>

    <div>
      <h2 class="text-sm font-bold text-slate-400 uppercase tracking-wider mb-3 px-1 flex items-center justify-between">
        <span>연동 기기 실시간 모니터링 (Cloud Edge)</span>
        <span class="text-xs text-slate-500 font-normal">30초 자동 동기화</span>
      </h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4" id="deviceList"></div>
    </div>

    <div class="bg-slate-800 rounded-2xl p-5 border border-slate-700 space-y-4">
      <h3 class="font-bold text-sm text-slate-200 flex items-center gap-2">
        <i class="fa-solid fa-sliders text-indigo-400"></i> 스크린타임 설정
      </h3>
      <div class="flex flex-wrap items-center justify-between gap-4 text-sm">
        <div class="flex items-center gap-3">
          <label for="quotaInput" class="text-slate-300">하루 기본 할당 시간:</label>
          <div class="flex items-center gap-1">
            <input type="number" id="quotaInput" min="10" max="720" step="10" value="120" class="w-20 bg-slate-900 border border-slate-600 rounded-lg px-3 py-1.5 text-center text-white font-bold focus:outline-none focus:border-indigo-500">
            <span class="text-slate-400 text-xs">분</span>
          </div>
          <button onclick="updateQuota()" class="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-bold transition">
            변경 저장
          </button>
        </div>

        <button onclick="resetToday()" class="text-xs text-slate-400 hover:text-amber-400 transition flex items-center gap-1">
          <i class="fa-solid fa-rotate-left"></i> 오늘 사용량 초기화 (테스트용)
        </button>
      </div>
    </div>
  </main>

  <script>
    let currentData = null;

    async function fetchStatus() {
      const icon = document.getElementById('refreshIcon');
      if (icon) icon.classList.add('fa-spin');
      try {
        const res = await fetch('/api/status?child_id=child_1');
        if (!res.ok) throw new Error('Status load failed');
        currentData = await res.json();
        render();
      } catch (e) {
        console.error(e);
      } finally {
        if (icon) icon.classList.remove('fa-spin');
      }
    }

    function render() {
      if (!currentData || !currentData.child) return;
      const child = currentData.child;
      const totalAllowed = (child.daily_quota_minutes + child.bonus_minutes) * 60;
      const remainingSeconds = Math.max(0, totalAllowed - child.used_seconds);
      const remainingMinutes = Math.ceil(remainingSeconds / 60);
      const usedMinutes = Math.floor(child.used_seconds / 60);

      document.getElementById('childName').textContent = child.name;
      document.getElementById('dailyQuota').textContent = child.daily_quota_minutes + '분';
      document.getElementById('bonusTime').textContent = '+' + child.bonus_minutes + '분';
      document.getElementById('usedTime').textContent = usedMinutes + '분';

      const remEl = document.getElementById('remainingTime');
      remEl.textContent = remainingMinutes + '분';
      if (remainingMinutes <= 5 || child.force_locked) {
        remEl.className = 'text-4xl md:text-5xl font-black tracking-tight text-red-400';
      } else {
        remEl.className = 'text-4xl md:text-5xl font-black tracking-tight text-white';
      }

      const lockBadge = document.getElementById('lockBadge');
      const lockBtn = document.getElementById('lockToggleBtn');
      const lockBtnText = document.getElementById('lockBtnText');

      if (child.force_locked) {
        lockBadge.classList.remove('hidden');
        lockBtn.className = 'flex-1 md:flex-none px-4 py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition shadow-lg bg-emerald-600 hover:bg-emerald-500 text-white';
        lockBtn.innerHTML = '<i class="fa-solid fa-lock-open"></i><span>잠금 해제</span>';
      } else {
        lockBadge.classList.add('hidden');
        lockBtn.className = 'flex-1 md:flex-none px-4 py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition shadow-lg bg-red-600 hover:bg-red-500 text-white';
        lockBtn.innerHTML = '<i class="fa-solid fa-hand"></i><span>즉시 올스톱 잠금</span>';
      }

      const pct = totalAllowed > 0 ? Math.min(100, Math.round((child.used_seconds / totalAllowed) * 100)) : 100;
      const pBar = document.getElementById('progressBar');
      pBar.style.width = pct + '%';
      document.getElementById('progressText').textContent = pct + '% 소진';
      if (pct >= 90) {
        pBar.className = 'bg-red-500 h-full rounded-full transition-all duration-500';
      } else if (pct >= 70) {
        pBar.className = 'bg-amber-500 h-full rounded-full transition-all duration-500';
      } else {
        pBar.className = 'bg-indigo-500 h-full rounded-full transition-all duration-500';
      }

      const deviceContainer = document.getElementById('deviceList');
      deviceContainer.innerHTML = '';

      const devices = currentData.devices || [];
      devices.forEach(dev => {
        let iconClass = 'fa-mobile-screen-button';
        let typeBadge = '안드로이드 폰';
        if (dev.type === 'android_tablet') {
          iconClass = 'fa-tablet-screen-button';
          typeBadge = '태블릿';
        } else if (dev.type === 'windows_pc') {
          iconClass = 'fa-desktop';
          typeBadge = '윈도우 PC';
        }

        let statusBadge = '<span class="text-xs text-slate-500">⚪ 오프라인</span>';
        let borderClass = 'border-slate-800';

        if (dev.status === 'online') {
          statusBadge = '<span class="text-xs text-emerald-400 font-bold flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-emerald-400 pulse-online"></span> 사용 중</span>';
          borderClass = 'border-emerald-500/40 bg-slate-800/90 shadow-md shadow-emerald-950/30';
        } else if (dev.status === 'idle') {
          statusBadge = '<span class="text-xs text-amber-400 font-semibold">🟡 유휴(자리비움)</span>';
          borderClass = 'border-amber-500/30';
        } else if (dev.status === 'locked') {
          statusBadge = '<span class="text-xs text-red-400 font-bold">🔒 화면 잠김</span>';
          borderClass = 'border-red-500/40';
        }

        const card = document.createElement('div');
        card.className = \`bg-slate-800/80 rounded-xl p-4 border \${borderClass} transition relative overflow-hidden flex flex-col justify-between\`;
        card.innerHTML = \`
          <div>
            <div class="flex items-start justify-between mb-2">
              <div class="flex items-center gap-2.5">
                <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center text-slate-300">
                  <i class="fa-solid \${iconClass}"></i>
                </div>
                <div>
                  <h4 class="font-bold text-sm text-white">\${dev.name}</h4>
                  <span class="text-[11px] text-slate-400">\${typeBadge}</span>
                </div>
              </div>
              <div>\${statusBadge}</div>
            </div>
            <div class="mt-3 bg-slate-900/60 rounded-lg p-2.5 border border-slate-700/50">
              <div class="text-[11px] text-slate-400">현재 활성 프로그램 / 앱:</div>
              <div class="text-xs font-semibold text-slate-200 truncate mt-0.5" title="\${dev.current_app || '없음'}">
                \${dev.current_app || '대기 중'}
              </div>
            </div>
          </div>
          <div class="mt-3 text-[11px] text-slate-500 flex justify-between items-center">
            <span>ID: \${dev.id}</span>
            <span>최근 동기화: \${formatTime(dev.last_seen)}</span>
          </div>
        \`;
        deviceContainer.appendChild(card);
      });
    }

    function formatTime(isoStr) {
      if (!isoStr || isoStr.startsWith('0001')) return '기록 없음';
      const d = new Date(isoStr);
      return d.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    }

    async function toggleLock() {
      if (!currentData || !currentData.child) return;
      const targetState = !currentData.child.force_locked;
      await fetch('/api/lock', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ child_id: 'child_1', lock: targetState })
      });
      fetchStatus();
    }

    async function addBonus(mins) {
      await fetch('/api/bonus', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ child_id: 'child_1', minutes: mins })
      });
      fetchStatus();
    }

    async function updateQuota() {
      const val = parseInt(document.getElementById('quotaInput').value, 10);
      if (isNaN(val) || val <= 0) return alert('올바른 분(시간)을 입력하세요');
      await fetch('/api/quota', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ child_id: 'child_1', minutes: val })
      });
      fetchStatus();
    }

    async function resetToday() {
      if (!confirm('오늘의 사용 시간을 초기화하시겠습니까? (테스트용)')) return;
      await fetch('/api/reset?child_id=child_1');
      fetchStatus();
    }

    fetchStatus();
    setInterval(fetchStatus, 2000);
  </script>
</body>
</html>`;
