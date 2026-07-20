/* =========================================================
 * Blink 落地页脚本
 * - 从 GitHub API 拉取最新 Release（含预发布），自动填充「一键下载」按钮
 * - 部署到 *.github.io 时自动识别仓库；本地/其他域名回退到真实仓库
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

  function setButton(btn, url, label) {
    if (!btn) return;
    btn.href = url || "#";
    btn.removeAttribute("data-loading");
    const t = btn.querySelector(".btn-text");
    if (t && label) t.textContent = label;
  }

  async function initDownload() {
    const repo = getRepo();
    // 用 releases 列表（含预发布），按创建时间倒序，第一条即最新
    const apiUrl = `https://api.github.com/repos/${repo.owner}/${repo.repo}/releases?per_page=1`;
    const repoUrl = `https://github.com/${repo.owner}/${repo.repo}`;
    const releasesUrl = repoUrl + "/releases";

    const primaryBtns = [document.getElementById("download-btn"), document.getElementById("download-btn-2")];
    const intelLink = document.getElementById("download-intel");
    const versionTag = document.getElementById("version-tag");
    const releasesLink = document.getElementById("releases-link");

    if (releasesLink) releasesLink.href = releasesUrl;
    primaryBtns.forEach((b) => b && (b.href = releasesUrl));
    // 自动把占位 github.com 链接（导航/页脚）指向真实仓库
    document
      .querySelectorAll('a[href="https://github.com"]')
      .forEach((a) => (a.href = repoUrl));

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
      const intel = findAsset(data.assets, "-amd64.dmg") || findAsset(data.assets, "-x64.dmg");

      if (arm) {
        primaryBtns.forEach((b) => setButton(b, arm.browser_download_url, "下载 Blink"));
      }
      if (intel && intelLink) {
        intelLink.href = intel.browser_download_url;
        intelLink.style.display = "";
        intelLink.textContent = "Intel 版本";
      } else if (intelLink) {
        intelLink.style.display = "none";
      }
    } catch (err) {
      // 网络/限流失败：保留按钮指向 Releases 页面，版本标签降级
      if (versionTag) versionTag.textContent = "免费 · 开源";
      primaryBtns.forEach((b) => b && b.removeAttribute("data-loading"));
      if (intelLink) intelLink.style.display = "none";
      console.warn("获取最新 Release 失败，已降级到 Releases 页面：", err);
    }
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
    initReveal();
  });
})();
