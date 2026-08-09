"use strict";

const COURSE_FILES = ["data/day001.json"];
const STORAGE_KEY = "ml-growth:v2:progress";
const LEGACY_STORAGE_KEY = "ml-growth:v1:progress";
const WEEKDAYS = ["一", "二", "三", "四", "五", "六", "日"];
const views = {
  home: document.querySelector("#home-view"),
  lesson: document.querySelector("#lesson-view"),
  error: document.querySelector("#error-view")
};

let courses = [];
let calendarCursor = new Date(new Date().getFullYear(), new Date().getMonth(), 1);
let activeCourseDay = null;
let selectedRating = 0;

function emptyProgress() {
  return { version: 2, completedDays: {}, quizResults: {}, dailyRatings: {}, dailyStudySeconds: {}, activeStudy: null };
}

function readProgress() {
  try {
    const current = JSON.parse(localStorage.getItem(STORAGE_KEY));
    if (current && typeof current === "object") return { ...emptyProgress(), ...current };
    const legacy = JSON.parse(localStorage.getItem(LEGACY_STORAGE_KEY));
    if (legacy && typeof legacy === "object") {
      const completedDays = Object.fromEntries(Object.entries(legacy.completedDays || {}).map(([day, item]) => [day, {
        ...item,
        completed: item.completed !== false,
        completedDate: item.completedDate || localDateKey(new Date(item.completedAt || Date.now()))
      }]));
      const migrated = { ...emptyProgress(), completedDays };
      saveProgress(migrated);
      return migrated;
    }
  } catch { /* 使用干净的数据结构 */ }
  return emptyProgress();
}

function saveProgress(progress) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ ...emptyProgress(), ...progress, version: 2, updatedAt: new Date().toISOString() }));
}

function localDateKey(date = new Date()) {
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 10);
}

function formatDuration(totalSeconds) {
  const minutes = Math.floor(Math.max(0, totalSeconds) / 60);
  if (minutes < 60) return `${minutes} 分钟`;
  return `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分钟`;
}

function escapeHtml(value = "") {
  return String(value).replace(/[&<>'"]/g, character => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;"
  })[character]);
}

function showView(name) {
  Object.entries(views).forEach(([key, element]) => { element.hidden = key !== name; });
  window.scrollTo({ top: 0, behavior: "smooth" });
}

async function loadCourses() {
  const results = await Promise.all(COURSE_FILES.map(async file => {
    const response = await fetch(file);
    if (!response.ok) throw new Error(`课程文件加载失败：${file}`);
    return response.json();
  }));
  courses = results.sort((a, b) => a.day - b.day);
}

function countCompletedOn(dateKey, progress) {
  return Object.values(progress.completedDays).filter(item => item.completed && item.completedDate === dateKey).length;
}

function isLearningDay(dateKey, progress) {
  return courses.length > 0 && countCompletedOn(dateKey, progress) >= courses.length;
}

function calculateStreak(progress) {
  let cursor = new Date();
  let streak = 0;
  while (isLearningDay(localDateKey(cursor), progress)) {
    streak += 1;
    cursor.setDate(cursor.getDate() - 1);
  }
  return streak;
}

function renderHome() {
  const progress = readProgress();
  const completedCount = courses.filter(course => progress.completedDays[course.day]?.completed).length;
  const percentage = courses.length ? (completedCount / courses.length) * 100 : 0;
  document.querySelector("#progress-label").textContent = `已完成 ${completedCount} / ${courses.length}`;
  document.querySelector("#progress-bar").style.width = `${percentage}%`;

  const nextCourse = courses.find(course => !progress.completedDays[course.day]?.completed) || courses[0];
  const continueButton = document.querySelector("#continue-button");
  if (nextCourse) {
    continueButton.href = `#/day/${nextCourse.day}`;
    continueButton.firstChild.textContent = completedCount === courses.length ? "再次学习 " : `继续 Day ${nextCourse.day} `;
  }

  document.querySelector("#task-list").innerHTML = courses.map(course => {
    const complete = Boolean(progress.completedDays[course.day]?.completed);
    const quiz = progress.quizResults[course.day];
    return `<a class="task-card ${complete ? "is-complete" : ""}" href="#/day/${course.day}" aria-label="${complete ? "已完成" : "未完成"}：Day ${course.day} ${escapeHtml(course.title)}">
      <span class="day-box"><small>DAY</small><b>${String(course.day).padStart(2, "0")}</b></span>
      <span class="task-copy"><p>${escapeHtml(course.chapter)} · 约 ${escapeHtml(course.durationMinutes)} 分钟</p><h3>${escapeHtml(course.title)}</h3><ul>${course.tasks.map(task => `<li>${escapeHtml(task)}</li>`).join("")}</ul></span>
      <span class="task-status">${complete ? "✓ 已完成" : quiz ? `测验 ${quiz.score}%` : "○ 未完成"}</span>
    </a>`;
  }).join("");
  renderLearningRecord(progress);
}

function renderLearningRecord(progress) {
  const today = localDateKey();
  const totalSeconds = Object.values(progress.dailyStudySeconds).reduce((sum, seconds) => sum + Number(seconds || 0), 0);
  const todaySeconds = Number(progress.dailyStudySeconds[today] || 0) + getLiveStudySeconds(progress, today);
  document.querySelector("#today-time").textContent = formatDuration(todaySeconds);
  document.querySelector("#total-time").textContent = formatDuration(totalSeconds + getLiveStudySeconds(progress));
  const streak = calculateStreak(progress);
  document.querySelector("#streak-count").textContent = `${streak} 天`;
  document.querySelector("#streak-detail").textContent = streak ? "保持节奏，明天再来一点。" : "完成当天全部任务即可点亮";
  renderCalendar(progress);
}

function getLiveStudySeconds(progress, dateKey) {
  const active = progress.activeStudy;
  if (!active || document.hidden) return 0;
  if (dateKey && active.date !== dateKey) return 0;
  return Math.max(0, Math.floor((Date.now() - new Date(active.startedAt).getTime()) / 1000));
}

function renderCalendar(progress) {
  const year = calendarCursor.getFullYear();
  const month = calendarCursor.getMonth();
  document.querySelector("#calendar-title").textContent = `${year} 年 ${month + 1} 月`;
  const firstDay = new Date(year, month, 1);
  const leading = (firstDay.getDay() + 6) % 7;
  const days = new Date(year, month + 1, 0).getDate();
  const today = localDateKey();
  const cells = [];
  for (let index = 0; index < leading; index += 1) cells.push('<span class="calendar-day blank" aria-hidden="true"></span>');
  for (let day = 1; day <= days; day += 1) {
    const date = new Date(year, month, day);
    const key = localDateKey(date);
    const complete = isLearningDay(key, progress);
    const isToday = key === today;
    cells.push(`<span class="calendar-day ${complete ? "complete" : ""} ${isToday ? "today" : ""}" aria-label="${key}${complete ? "，已完成学习" : "，未完成学习"}"><b>${day}</b>${complete ? '<i aria-hidden="true">✓</i>' : ""}</span>`);
  }
  document.querySelector("#calendar-grid").innerHTML = cells.join("");
}

function renderQuiz(course, result) {
  const quiz = course.quiz?.[0];
  if (!quiz) return "";
  if (result) {
    return `<section class="lesson-section quiz-section"><h2>6. 小测试</h2><div class="quiz-result ${result.correct ? "correct" : "incorrect"}"><strong>${result.correct ? "回答正确！" : "这题先记下来。"}</strong><p>你的得分：${result.score}%</p><p>${escapeHtml(quiz.explanation)}</p><button class="secondary-button" type="button" data-action="retry-quiz">再试一次</button></div></section>`;
  }
  return `<section class="lesson-section quiz-section"><h2>6. 小测试</h2><p class="quiz-question">${escapeHtml(quiz.question)}</p><form id="quiz-form" class="quiz-form">${quiz.options.map((option, index) => `<label class="quiz-option"><input type="radio" name="quiz-answer" value="${index}"><span>${String.fromCharCode(65 + index)}</span>${escapeHtml(option)}</label>`).join("")}<button class="secondary-button" type="submit">提交答案</button></form></section>`;
}

function renderLesson(course) {
  const progress = readProgress();
  const completion = progress.completedDays[course.day];
  const complete = Boolean(completion?.completed);
  const sections = course.content;
  const rating = progress.dailyRatings[completion?.completedDate];
  document.querySelector("#lesson-content").innerHTML = `
    <header class="lesson-header"><p class="lesson-kicker">DAY ${String(course.day).padStart(2, "0")} · ${escapeHtml(course.chapter)}</p><h1 id="lesson-title">${escapeHtml(course.lessonTitle)}</h1><p class="lesson-summary">${escapeHtml(course.summary)}</p></header>
    <section class="lesson-section"><h2>1. 概念解释</h2><p>${escapeHtml(sections.concept)}</p><div class="analogy-card"><strong>直观类比</strong><p>${escapeHtml(sections.analogy)}</p></div></section>
    <section class="lesson-section"><h2>2. 数学解释</h2><p>${escapeHtml(sections.math.explanation)}</p><div class="formula">${escapeHtml(sections.math.formula)}</div></section>
    <section class="lesson-section"><h2>3. 工程案例</h2><p>${escapeHtml(sections.engineeringCase)}</p></section>
    <section class="lesson-section"><h2>4. Python / PyTorch 示例</h2><p>${escapeHtml(sections.code.explanation)}</p><pre><code>${escapeHtml(sections.code.source)}</code></pre></section>
    <section class="lesson-section"><h2>5. 常见错误</h2><ul class="mistakes">${sections.commonMistakes.map(item => `<li>${escapeHtml(item)}</li>`).join("")}</ul></section>
    <section class="lesson-section"><h2>本节学习目标</h2><ul class="lesson-tasks">${course.tasks.map(item => `<li>${escapeHtml(item)}</li>`).join("")}</ul></section>
    ${renderQuiz(course, progress.quizResults[course.day])}
    <footer class="lesson-footer"><p>${complete ? `你已经完成本节${rating ? `，今天的掌握度：${"★".repeat(rating)}${"☆".repeat(5 - rating)}。` : "，可随时回来复习。"}` : "确认理解核心概念后，记录今天的这一步。"}</p><button id="complete-button" class="primary-button ${complete ? "completed" : ""}" type="button">${complete ? "✓ 已完成学习" : "完成学习"}</button></footer>`;
  document.querySelector("#complete-button").addEventListener("click", () => completeLesson(course));
  document.querySelector("#quiz-form")?.addEventListener("submit", event => submitQuiz(event, course));
  document.querySelector("[data-action='retry-quiz']")?.addEventListener("click", () => { const data = readProgress(); delete data.quizResults[course.day]; saveProgress(data); renderLesson(course); });
}

function submitQuiz(event, course) {
  event.preventDefault();
  const choice = new FormData(event.currentTarget).get("quiz-answer");
  if (choice === null) { showToast("请先选择一个答案。"); return; }
  const correct = Number(choice) === course.quiz[0].answerIndex;
  const progress = readProgress();
  progress.quizResults[course.day] = { selectedIndex: Number(choice), correct, score: correct ? 100 : 0, answeredAt: new Date().toISOString() };
  saveProgress(progress);
  renderLesson(course);
}

function completeLesson(course) {
  const progress = readProgress();
  const date = localDateKey();
  const wasComplete = progress.completedDays[course.day]?.completed;
  progress.completedDays[course.day] = { completed: true, completedAt: new Date().toISOString(), completedDate: date };
  saveProgress(progress);
  renderLesson(course);
  if (!wasComplete && !progress.dailyRatings[date]) openRatingDialog(date);
  else showToast(`Day ${course.day} 已完成，今天的基础更牢了一点。`);
}

function beginStudy(courseDay) {
  if (document.hidden) return;
  const progress = readProgress();
  if (progress.activeStudy?.day === courseDay) return;
  pauseStudy(progress);
  progress.activeStudy = { day: courseDay, date: localDateKey(), startedAt: new Date().toISOString() };
  saveProgress(progress);
}

function pauseStudy(progress = readProgress()) {
  const active = progress.activeStudy;
  if (!active) return;
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(active.startedAt).getTime()) / 1000));
  progress.dailyStudySeconds[active.date] = Number(progress.dailyStudySeconds[active.date] || 0) + seconds;
  progress.activeStudy = null;
  saveProgress(progress);
}

function openRatingDialog() {
  selectedRating = 0;
  document.querySelector("#rating-dialog").hidden = false;
  updateStarRating();
  document.querySelector("#star-rating button").focus();
}

function closeRatingDialog() { document.querySelector("#rating-dialog").hidden = true; }

function updateStarRating() {
  document.querySelectorAll("#star-rating button").forEach(button => {
    const on = Number(button.dataset.rating) <= selectedRating;
    button.classList.toggle("selected", on);
    button.setAttribute("aria-checked", String(Number(button.dataset.rating) === selectedRating));
  });
  const labels = ["选择 1–5 星", "还需要重新理解基础概念", "有一点感觉，但需要复习", "基本理解，建议再做一次练习", "理解清晰，可以继续前进", "掌握得很好，准备迎接下一节！"];
  document.querySelector("#rating-hint").textContent = labels[selectedRating];
  document.querySelector("#save-rating").disabled = selectedRating === 0;
}

function saveRating() {
  if (!selectedRating) return;
  const progress = readProgress();
  progress.dailyRatings[localDateKey()] = selectedRating;
  saveProgress(progress);
  closeRatingDialog();
  renderHome();
  showToast(`已记录 ${"★".repeat(selectedRating)}，这会帮助之后安排复习。`);
}

function showToast(message) {
  const toast = document.querySelector("#toast");
  toast.textContent = message;
  toast.classList.add("show");
  window.setTimeout(() => toast.classList.remove("show"), 2800);
}

function showError(message) { document.querySelector("#error-message").textContent = message; showView("error"); }

function route() {
  pauseStudy();
  const match = window.location.hash.match(/^#\/day\/(\d+)$/);
  if (!match || window.location.hash === "#/") { activeCourseDay = null; renderHome(); showView("home"); document.title = "ML Growth · 机器学习成长系统"; return; }
  const course = courses.find(item => item.day === Number(match[1]));
  if (!course) { showError("该课程尚未加入学习计划，请返回首页选择现有任务。"); return; }
  activeCourseDay = course.day;
  renderLesson(course); showView("lesson"); beginStudy(course.day); document.title = `${course.lessonTitle} · ML Growth`;
}

function bindEvents() {
  document.querySelector("#previous-month").addEventListener("click", () => { calendarCursor.setMonth(calendarCursor.getMonth() - 1); renderCalendar(readProgress()); });
  document.querySelector("#next-month").addEventListener("click", () => { calendarCursor.setMonth(calendarCursor.getMonth() + 1); renderCalendar(readProgress()); });
  document.querySelectorAll("#star-rating button").forEach(button => button.addEventListener("click", () => { selectedRating = Number(button.dataset.rating); updateStarRating(); }));
  document.querySelector("#save-rating").addEventListener("click", saveRating);
  document.querySelector("#skip-rating").addEventListener("click", closeRatingDialog);
  document.addEventListener("visibilitychange", () => { if (document.hidden) pauseStudy(); else if (activeCourseDay) beginStudy(activeCourseDay); });
  window.addEventListener("beforeunload", () => pauseStudy());
  window.addEventListener("hashchange", route);
  window.setInterval(() => { if (!document.hidden && !views.home.hidden) renderLearningRecord(readProgress()); }, 30000);
}

async function start() {
  try { await loadCourses(); bindEvents(); route(); } catch (error) { showError("课程数据无法加载。请按照 README 使用本地服务器打开网站，不要直接双击 index.html。"); console.error(error); }
}

start();
