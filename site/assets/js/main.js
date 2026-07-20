/* =========================================================
 * Blink 落地页脚本
 * - 从 GitHub API 拉取最新 Release（含预发布），自动填充「一键下载」按钮
 * - 部署到 *.github.io 时自动识别仓库；本地/其他域名回退到真实仓库
 * - 分体下载按钮：默认 Apple Silicon，可下拉切换 Intel
 * - API 失败时优雅降级到 Releases 页面
 *
 * 说明：本项目目前只发 beta（pre-release），而 /releases/latest 只返回
 * 非预发布版本，会 404。因此这里用 /releases 列表接口，取最新一条。
 * ========================================================= */

// 如需覆盖仓库（例如绑定自定义域名时自动识别失效），可取消下面注释：
// window.BLINK_REPO = { owner: "wrm244", repo: "Blink" };

// 真实仓库（本地预览 / 非 github.io 域名时的回退值）
const REAL_REPO = { owner: "wrm244", repo: "Blink" };

(function () {
  "use strict";

  const ARCH_LABELS = { arm64: "Apple Silicon", amd64: "Intel Mac" };

  function getRepo() {
    if (window.BLINK_REPO && window.BLINK_REPO.owner && window.BLINK_REPO.repo) {
      return window.BLINK_REPO;
    }
    // 部署到 GitHub Pages 项目页：owner.github.io/<repo>/
    const parts = window.location.pathname.split("/").filter(Boolean);
    const hostMatch = window.location.hostname.match(/^([^.]+)\.github\.io$/);
    if (hostMatch && parts.length > 0) {
      return { owner: hostMatch[1], repo: parts[0] };
    }
    return REAL_REPO;
  }

  function findAsset(assets, suffix) {
    if (!assets || !assets.length) return null;
    return assets.find((a) => a.name.toLowerCase().endsWith(suffix)) || null;
  }

  function closeAllMenus() {
    document.querySelectorAll(".dl.open").forEach((dl) => {
      dl.classList.remove("open");
      const t = dl.querySelector(".dl-toggle");
      if (t) t.setAttribute("aria-expanded", "false");
    });
  }

  // 设置某个分体按钮当前选中的架构，并刷新主按钮指向
  function setDlArch(dl, arch, fallbackUrl) {
    const url = dl._urls && dl._urls[arch] ? dl._urls[arch] : fallbackUrl || "#";
    const main = dl.querySelector(".dl-main");
    const archEl = dl.querySelector(".dl-arch");
    if (main) main.href = url;
    if (archEl) archEl.textContent = ARCH_LABELS[arch] || "";
    dl.querySelectorAll(".dl-item").forEach((it) =>
      it.classList.toggle("is-active", it.dataset.arch === arch)
    );
    dl._current = arch;
  }

  async function initDownload() {
    const repo = getRepo();
    // 用 releases 列表（含预发布），按创建时间倒序，第一条即最新
    const apiUrl = `https://api.github.com/repos/${repo.owner}/${repo.repo}/releases?per_page=1`;
    const repoUrl = `https://github.com/${repo.owner}/${repo.repo}`;
    const releasesUrl = repoUrl + "/releases";

    const versionTag = document.getElementById("version-tag");
    const releasesLink = document.getElementById("releases-link");
    const dls = Array.from(document.querySelectorAll(".dl"));

    if (releasesLink) releasesLink.href = releasesUrl;
    // 自动把占位 github.com 链接（导航/页脚）指向真实仓库
    document
      .querySelectorAll('a[href="https://github.com"]')
      .forEach((a) => (a.href = repoUrl));

    // 先降级：默认 Apple Silicon 指向 Releases 页，等 API 返回再替换
    dls.forEach((dl) => setDlArch(dl, "arm64", releasesUrl));

    try {
      const res = await fetch(apiUrl, {
        headers: { Accept: "application/vnd.github+json" },
      });
      if (!res.ok) throw new Error("HTTP " + res.status);
      const list = await res.json();
      const data = Array.isArray(list) ? list[0] : list;
      if (!data) throw new Error("no release");

      const tag = data.tag_name || data.name || "";
      if (versionTag) versionTag.textContent = "最新版本 " + tag;

      const arm = findAsset(data.assets, "-arm64.dmg") || findAsset(data.assets, ".dmg");
      const intel =
        findAsset(data.assets, "-amd64.dmg") || findAsset(data.assets, "-x64.dmg");

      dls.forEach((dl) => {
        dl._urls = {
          arm64: arm ? arm.browser_download_url : releasesUrl,
          amd64: intel ? intel.browser_download_url : releasesUrl,
        };
        // Intel 不存在则禁用该项
        const intelItem = dl.querySelector('.dl-item[data-arch="amd64"]');
        if (!intel && intelItem) {
          intelItem.classList.add("is-disabled");
          intelItem.disabled = true;
        }
        // 保持当前选中架构，刷新其下载地址
        setDlArch(dl, dl._current || "arm64", releasesUrl);
      });
    } catch (err) {
      // 网络/限流失败：保留按钮指向 Releases 页面，版本标签降级
      if (versionTag) versionTag.textContent = "免费 · 开源";
      console.warn("获取最新 Release 失败，已降级到 Releases 页面：", err);
    }
  }

  function initSplitButtons() {
    document.querySelectorAll(".dl").forEach((dl) => {
      const toggle = dl.querySelector(".dl-toggle");
      if (toggle) {
        toggle.addEventListener("click", function (e) {
          e.stopPropagation();
          const willOpen = !dl.classList.contains("open");
          closeAllMenus();
          if (willOpen) {
            dl.classList.add("open");
            toggle.setAttribute("aria-expanded", "true");
          }
        });
      }
      dl.querySelectorAll(".dl-item").forEach((item) => {
        item.addEventListener("click", function () {
          if (item.disabled) return;
          setDlArch(dl, item.dataset.arch, null);
          dl.classList.remove("open");
          const t = dl.querySelector(".dl-toggle");
          if (t) t.setAttribute("aria-expanded", "false");
        });
      });
    });

    // 点击空白处 / 按 Esc 关闭所有菜单
    document.addEventListener("click", function (e) {
      if (!e.target.closest(".dl")) closeAllMenus();
    });
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") closeAllMenus();
    });
  }

  // 滚动揭示动画
  function initReveal() {
    const els = document.querySelectorAll(".reveal");
    if (!("IntersectionObserver" in window)) {
      els.forEach((e) => e.classList.add("in"));
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            e.target.classList.add("in");
            io.unobserve(e.target);
          }
        });
      },
      { threshold: 0.12 }
    );
    els.forEach((e) => io.observe(e));
  }

  document.addEventListener("DOMContentLoaded", function () {
    initDownload();
    initSplitButtons();
    initReveal();
  });
})();
