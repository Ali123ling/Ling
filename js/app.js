"use strict";

const COURSE_FILES = ["data/day001.json", "data/day002.json"];
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
let activeUnitIndex = 0;
let selectedRating = 0;

function emptyProgress() {
  return { version: 2, completedDays: {}, quizResults: {}, dailyRatings: {}, dailyStudySeconds: {}, courseStudySeconds: {}, activeStudy: null };
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
  return Object.values(progress.completedDays).filter(item => isQualifiedCompletion(item) && item.completedDate === dateKey).length;
}

function isQualifiedCompletion(completion) {
  return Boolean(completion?.completed && completion.validationMethod);
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
  const completedCount = courses.filter(course => isQualifiedCompletion(progress.completedDays[course.day])).length;
  const percentage = courses.length ? (completedCount / courses.length) * 100 : 0;
  document.querySelector("#progress-label").textContent = `已完成 ${completedCount} / ${courses.length}`;
  document.querySelector("#progress-bar").style.width = `${percentage}%`;

  const nextCourse = courses.find(course => !isQualifiedCompletion(progress.completedDays[course.day])) || courses[0];
  const continueButton = document.querySelector("#continue-button");
  if (nextCourse) {
    continueButton.href = `#/day/${nextCourse.day}`;
    continueButton.firstChild.textContent = completedCount === courses.length ? "再次学习 " : `继续 Day ${nextCourse.day} `;
  }

  document.querySelector("#task-list").innerHTML = courses.map(course => {
    const complete = isQualifiedCompletion(progress.completedDays[course.day]);
    const quiz = progress.quizResults[course.day];
    const currentQuiz = isQuizFullyAnswered(course, quiz) ? quiz : null;
    return `<a class="task-card ${complete ? "is-complete" : ""}" href="#/day/${course.day}" aria-label="${complete ? "已完成" : "未完成"}：Day ${course.day} ${escapeHtml(course.title)}">
      <span class="day-box"><small>DAY</small><b>${String(course.day).padStart(2, "0")}</b></span>
      <span class="task-copy"><p>${escapeHtml(course.chapter)} · 约 ${escapeHtml(course.durationMinutes)} 分钟</p><h3>${escapeHtml(course.title)}</h3><ul>${course.tasks.map(task => `<li>${escapeHtml(task)}</li>`).join("")}</ul></span>
      <span class="task-status">${complete ? "✓ 已完成" : currentQuiz ? `测验 ${currentQuiz.score}%` : "○ 未完成"}</span>
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

function renderUnits(course, result) {
  const units = course.units || [];
  activeUnitIndex = Math.max(0, Math.min(activeUnitIndex, units.length - 1));
  const unit = units[activeUnitIndex];
  if (!unit) return "";
  const number = activeUnitIndex + 1;
  return `<section class="lesson-unit">
    <header class="unit-header"><span class="unit-number">${String(number).padStart(2, "0")}</span><div><div class="unit-label-row"><p class="eyebrow">今日小节 ${number} / ${units.length}</p><nav class="unit-navigation" aria-label="切换学习小节"><button type="button" class="unit-nav-button" data-unit-step="-1" aria-label="上一节" ${number === 1 ? "disabled" : ""}>←</button><button type="button" class="unit-nav-button" data-unit-step="1" aria-label="下一节" ${number === units.length ? "disabled" : ""}>→</button></nav></div><h2>${escapeHtml(unit.title)}</h2><p>${escapeHtml(unit.goal)}</p></div></header>
    <div class="unit-block"><h3>先用直觉理解</h3><p>${escapeHtml(unit.concept)}</p><div class="analogy-card"><strong>工程类比</strong><p>${escapeHtml(unit.analogy)}</p></div></div>
    <div class="unit-block"><h3>数学上怎么表示？</h3><p>${escapeHtml(unit.math.explanation)}</p><div class="formula">${escapeHtml(unit.math.formula)}</div></div>
    <div class="unit-block"><h3>用 PyTorch 看一眼</h3><p>${escapeHtml(unit.code.explanation)}</p><pre><code>${escapeHtml(unit.code.source)}</code></pre></div>
    <div class="unit-block"><h3>容易踩的坑</h3><ul class="mistakes">${unit.commonMistakes.map(item => `<li>${escapeHtml(item)}</li>`).join("")}</ul></div>
    ${renderUnitQuiz(course, activeUnitIndex, result)}
  </section>`;
}

function getCourseStudySeconds(progress, courseDay) {
  const saved = Number(progress.courseStudySeconds?.[courseDay] || 0);
  const active = progress.activeStudy;
  const live = active?.day === courseDay && !document.hidden ? Math.max(0, Math.floor((Date.now() - new Date(active.startedAt).getTime()) / 1000)) : 0;
  return saved + live;
}

function quizCount(course) {
  return (course.units || []).filter(unit => unit.quiz).length;
}

function getUnitAnswer(course, result, unitIndex) {
  const answer = result?.answers?.[unitIndex];
  if (answer && typeof answer === "object") return answer;
  if (Array.isArray(result?.answers) && result.answers[unitIndex] !== undefined) {
    const selectedIndex = Number(result.answers[unitIndex]);
    return { selectedIndex, correct: selectedIndex === course.units[unitIndex]?.quiz?.answerIndex };
  }
  return null;
}

function getAnswerMap(course, result) {
  return Object.fromEntries((course.units || []).map((_, index) => [index, getUnitAnswer(course, result, index)]).filter(([, answer]) => answer));
}

function buildQuizResult(course, answers) {
  const answeredCount = Object.keys(answers).length;
  const correctCount = Object.values(answers).filter(answer => answer.correct).length;
  const total = quizCount(course);
  return { answers, questionCount: total, answeredCount, correctCount, correct: total > 0 && answeredCount === total && correctCount === total, score: total ? Math.round((correctCount / total) * 100) : 0, answeredAt: new Date().toISOString() };
}

function isQuizFullyAnswered(course, result) {
  return Boolean(result && Object.keys(getAnswerMap(course, result)).length === quizCount(course));
}

function isPerfectQuiz(course, result) {
  const total = quizCount(course);
  const answers = Object.values(getAnswerMap(course, result));
  return Boolean(total && answers.length === total && answers.every(answer => answer.correct));
}

function canCompleteCourse(course, progress) {
  return isPerfectQuiz(course, progress.quizResults[course.day]) || getCourseStudySeconds(progress, course.day) >= 3600;
}

function renderCompletionFooter(course, progress, complete, rating) {
  const eligible = canCompleteCourse(course, progress);
  const quiz = progress.quizResults[course.day];
  const correctCount = Object.values(getAnswerMap(course, quiz)).filter(answer => answer.correct).length;
  const quizProgress = isPerfectQuiz(course, quiz) ? "三题全对 ✓" : `${correctCount} / ${quizCount(course)} 题正确`;
  const minutes = Math.min(60, Math.floor(getCourseStudySeconds(progress, course.day) / 60));
  if (complete) return `<footer class="lesson-footer"><p>你已经完成今天三小节${rating ? `，今天的掌握度：${"★".repeat(rating)}${"☆".repeat(5 - rating)}。` : "，可随时回来复习。"}</p><button id="complete-button" class="primary-button completed" type="button">✓ 已完成今日学习</button></footer>`;
  if (eligible) return `<footer class="lesson-footer"><p>完成条件已达成，现在可以记录今日学习。</p><button id="complete-button" class="primary-button" type="button">完成今日三小节</button></footer>`;
  return `<footer class="lesson-footer completion-gate"><p><strong>完成按钮将在满足任一条件后出现：</strong>三题全对，或本课程累计学习满 60 分钟。</p><div class="gate-progress"><span>测试：${quizProgress}</span><span id="course-time-progress">学习：${minutes} / 60 分钟</span></div></footer>`;
}

function renderUnitQuiz(course, unitIndex, result) {
  const quiz = course.units[unitIndex]?.quiz;
  if (!quiz) return "";
  const answer = getUnitAnswer(course, result, unitIndex);
  if (answer) return `<section class="unit-quiz"><p class="eyebrow">小节练习</p><h3>检验这一节</h3><div class="quiz-result ${answer.correct ? "correct" : "incorrect"}"><strong>${answer.correct ? "回答正确！" : "再回看一下上面的知识点。"}</strong><p>${escapeHtml(quiz.explanation)}</p><button class="secondary-button" type="button" data-retry-unit="${unitIndex}">重做本题</button></div></section>`;
  return `<section class="unit-quiz"><p class="eyebrow">小节练习</p><h3>检验这一节</h3><form class="unit-quiz-form quiz-form" data-unit-index="${unitIndex}"><fieldset class="quiz-group"><legend class="quiz-question">${escapeHtml(quiz.question)}</legend>${quiz.options.map((option, optionIndex) => `<label class="quiz-option"><input type="radio" name="unit-quiz-answer" value="${optionIndex}"><span>${String.fromCharCode(65 + optionIndex)}</span>${escapeHtml(option)}</label>`).join("")}</fieldset><button class="secondary-button" type="submit">提交本题</button></form></section>`;
}

function renderLesson(course) {
  const progress = readProgress();
  const completion = progress.completedDays[course.day];
  const complete = isQualifiedCompletion(completion);
  const rating = progress.dailyRatings[completion?.completedDate];
  document.querySelector("#lesson-content").innerHTML = `
    <header class="lesson-header"><p class="lesson-kicker">DAY ${String(course.day).padStart(2, "0")} · ${escapeHtml(course.chapter)}</p><h1 id="lesson-title">${escapeHtml(course.lessonTitle)}</h1><p class="lesson-summary">${escapeHtml(course.summary)}</p></header>
    <section class="lesson-section lesson-roadmap"><h2>今天的三小节</h2><ul class="lesson-tasks">${course.tasks.map(item => `<li>${escapeHtml(item)}</li>`).join("")}</ul></section>
    ${renderUnits(course, progress.quizResults[course.day])}
    ${renderCompletionFooter(course, progress, complete, rating)}`;
  document.querySelector("#complete-button")?.addEventListener("click", () => completeLesson(course));
  document.querySelectorAll(".unit-quiz-form").forEach(form => form.addEventListener("submit", event => submitUnitQuiz(event, course)));
  document.querySelectorAll("[data-retry-unit]").forEach(button => button.addEventListener("click", () => retryUnitQuiz(course, Number(button.dataset.retryUnit))));
  document.querySelectorAll("[data-unit-step]").forEach(button => button.addEventListener("click", () => {
    activeUnitIndex += Number(button.dataset.unitStep);
    renderLesson(course);
    document.querySelector(".lesson-unit")?.scrollIntoView({ behavior: "smooth", block: "start" });
  }));
}

function submitUnitQuiz(event, course) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const selected = form.get("unit-quiz-answer");
  if (selected === null) { showToast("请先选择一个答案。"); return; }
  const unitIndex = Number(event.currentTarget.dataset.unitIndex);
  const question = course.units[unitIndex].quiz;
  const progress = readProgress();
  const answers = getAnswerMap(course, progress.quizResults[course.day]);
  answers[unitIndex] = { selectedIndex: Number(selected), correct: Number(selected) === question.answerIndex };
  progress.quizResults[course.day] = buildQuizResult(course, answers);
  saveProgress(progress);
  renderLesson(course);
}

function retryUnitQuiz(course, unitIndex) {
  const progress = readProgress();
  const answers = getAnswerMap(course, progress.quizResults[course.day]);
  delete answers[unitIndex];
  progress.quizResults[course.day] = buildQuizResult(course, answers);
  saveProgress(progress);
  renderLesson(course);
}

function completeLesson(course) {
  const progress = readProgress();
  if (!isQualifiedCompletion(progress.completedDays[course.day]) && !canCompleteCourse(course, progress)) {
    showToast("请先答对全部三题，或累计学习满 60 分钟。");
    return;
  }
  const date = localDateKey();
  const wasComplete = isQualifiedCompletion(progress.completedDays[course.day]);
  progress.completedDays[course.day] = {
    completed: true,
    completedAt: new Date().toISOString(),
    completedDate: date,
    validationMethod: isPerfectQuiz(course, progress.quizResults[course.day]) ? "perfect-quiz" : "study-time"
  };
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
  progress.courseStudySeconds[active.day] = Number(progress.courseStudySeconds[active.day] || 0) + seconds;
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
  if (!match || window.location.hash === "#/") { activeCourseDay = null; activeUnitIndex = 0; renderHome(); showView("home"); document.title = "ML Growth · 机器学习成长系统"; return; }
  const course = courses.find(item => item.day === Number(match[1]));
  if (!course) { showError("该课程尚未加入学习计划，请返回首页选择现有任务。"); return; }
  if (activeCourseDay !== course.day) activeUnitIndex = 0;
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
  window.setInterval(() => {
    if (document.hidden) return;
    if (!views.home.hidden) renderLearningRecord(readProgress());
    if (!views.lesson.hidden && activeCourseDay) {
      const progress = readProgress();
      const course = courses.find(item => item.day === activeCourseDay);
      const time = document.querySelector("#course-time-progress");
      if (course && canCompleteCourse(course, progress) && time) renderLesson(course);
      else if (time) time.textContent = `学习：${Math.min(60, Math.floor(getCourseStudySeconds(progress, activeCourseDay) / 60))} / 60 分钟`;
    }
  }, 30000);
}

async function start() {
  try { await loadCourses(); bindEvents(); route(); } catch (error) { showError("课程数据无法加载。请按照 README 使用本地服务器打开网站，不要直接双击 index.html。"); console.error(error); }
}

start();
