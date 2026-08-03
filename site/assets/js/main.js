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

  const ARCH_LABELS = {
    arm64: "Apple Silicon",
    amd64: "Intel Mac",
    win64: "Windows · x64",
    winarm: "Windows · ARM64",
  };

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

  // 粗略识别访问者操作系统（mac / windows / other）。
  function detectOS() {
    const ua = navigator.userAgent || "";
    const platform = navigator.platform || "";
    const uaData = navigator.userAgentData;
    if (uaData && uaData.platform) {
      if (/Windows/i.test(uaData.platform)) return "windows";
      if (/Mac/i.test(uaData.platform)) return "mac";
    }
    if (/Win/i.test(platform) || /Windows/i.test(ua)) return "windows";
    if (/Mac/i.test(platform) || /Macintosh|Mac OS X/i.test(ua)) return "mac";
    return "other";
  }

  // 按访问者系统把对应平台的选项排到下拉菜单最前。
  function orderMenuForOS(dl, os) {
    const menu = dl.querySelector(".dl-menu");
    if (!menu) return;
    const order =
      os === "windows"
        ? ["win64", "winarm", "arm64", "amd64"]
        : ["arm64", "amd64", "win64", "winarm"];
    const items = Array.from(menu.querySelectorAll(".dl-item"));
    items
      .sort((a, b) => order.indexOf(a.dataset.arch) - order.indexOf(b.dataset.arch))
      .forEach((it) => menu.appendChild(it));
  }

  // 在当前系统的两个架构里挑一个「可用」的作为默认（资产缺失则回退到另一个）。
  function pickDefaultArch(dl, os) {
    const candidates = os === "windows" ? ["win64", "winarm"] : ["arm64", "amd64"];
    for (const a of candidates) {
      const item = dl.querySelector('.dl-item[data-arch="' + a + '"]');
      if (item && !item.disabled) return a;
    }
    return "arm64";
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
    const os = detectOS();

    if (releasesLink) releasesLink.href = releasesUrl;
    // 自动把占位 github.com 链接（导航/页脚）指向真实仓库
    document
      .querySelectorAll('a[href="https://github.com"]')
      .forEach((a) => (a.href = repoUrl));

    // 按访问者系统把对应平台安装包排到菜单最前，并预选为默认下载项
    const dfltArch = os === "windows" ? "win64" : "arm64";
    dls.forEach((dl) => {
      orderMenuForOS(dl, os);
      setDlArch(dl, dfltArch, releasesUrl);
    });

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
      // Windows 安装包命名：Blink-<版本>-windows-<arch>-setup.exe
      const winX64 =
        findAsset(data.assets, "-windows-amd64-setup.exe") ||
        findAsset(data.assets, "-amd64-setup.exe");
      const winArm =
        findAsset(data.assets, "-windows-arm64-setup.exe") ||
        findAsset(data.assets, "-arm64-setup.exe");

      dls.forEach((dl) => {
        dl._urls = {
          arm64: arm ? arm.browser_download_url : releasesUrl,
          amd64: intel ? intel.browser_download_url : releasesUrl,
          win64: winX64 ? winX64.browser_download_url : releasesUrl,
          winarm: winArm ? winArm.browser_download_url : releasesUrl,
        };
        // 缺失对应平台的资产时禁用该项，避免用户点到空地址
        const disableIfMissing = (arch, asset) => {
          const item = dl.querySelector('.dl-item[data-arch="' + arch + '"]');
          if (item && !asset) {
            item.classList.add("is-disabled");
            item.disabled = true;
          }
        };
        disableIfMissing("amd64", intel);
        disableIfMissing("win64", winX64);
        disableIfMissing("winarm", winArm);
        // 保持当前选中（按系统预置）；若其资产缺失则回退到同平台另一架构
        let sel = dl._current || (os === "windows" ? "win64" : "arm64");
        const selItem = dl.querySelector('.dl-item[data-arch="' + sel + '"]');
        if (selItem && selItem.disabled) sel = pickDefaultArch(dl, os);
        setDlArch(dl, sel, releasesUrl);
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
