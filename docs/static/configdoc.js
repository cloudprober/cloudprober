const setLang = async function (lang) {
  if (!lang) {
    lang = localStorage["preferred-language"];
  }
  if (!lang) {
    lang = "yaml";
  }

  const els = document.getElementsByClassName("select-content-lang");
  for (const e of els) {
    if (e.id == "content-lang-" + lang) {
      e.style.display = "block";
    } else {
      e.style.display = "none";
    }
  }
  localStorage["preferred-language"] = lang;

  const setLangButtons = document.getElementsByClassName("set-lang");
  for (const b of setLangButtons) {
    if (b.id == "set-lang-" + lang) {
      b.classList.add("btn-primary");
    } else {
      b.classList.remove("btn-primary");
    }
  }
};

const addButtonHandler = async function () {
  const setLangButtons = document.getElementsByClassName("set-lang");
  for (const b of setLangButtons) {
    b.addEventListener("click", async function () {
      const lang = b.id.replace("set-lang-", "");
      await setLang(lang);
    });
  }
};

const markConfigDocLinkActive = async function () {
  const configDocLinks = document.getElementById("config-doc-links");
  for (const a of configDocLinks.getElementsByTagName("a")) {
    if (a.href.replace(/\/$/, "") == window.location.href.replace(/\/$/, "")) {
      a.classList.add("config-link-active");
    }
  }
};

const subPackages = function (msgs) {
  const subPackages = {};
  for (const msg of msgs) {
    const parts = msg.split(".", 4);
    let sp = parts.slice(0, 2).join(".");
    if (parts.length > 3) {
      sp = parts.slice(0, 3).join(".");
    }
    if (!subPackages[sp]) {
      subPackages[sp] = [];
    }
    subPackages[sp].push(msg);
  }
  const ret = {
    pkgs: subPackages,
    names: [],
  };
  for (const sp in subPackages) {
    ret.names.push(sp);
  }
  ret.names.sort();
  return ret;
};

const addTOC = function () {
  const tocDivs = document.getElementsByClassName("docs-toc");
  if (!tocDivs || tocDivs.length == 0) {
    console.log("TOC div not found");
    return;
  }
  const tocTmpl = document.createElement("template");
  tocTmpl.innerHTML =
    '<div class="page-links d-none d-xl-block"><nav id="TableOfContents"><ul></ul></nav></div>';
  tocDivs[0].appendChild(tocTmpl.content.firstChild);
  const toc = document.getElementById("TableOfContents");

  const els = document.getElementsByClassName("select-content-lang");
  for (const el of els) {
    if (el.style.display == "none") continue;

    const h3s = el.getElementsByTagName("h3");
    if (h3s.length == 0) {
      continue;
    }

    let names = [];
    for (const h3 of h3s) {
      names.push(h3.innerText);
    }

    const subpkgs = subPackages(names);

    for (const name of subpkgs.names) {
      if (name.split(".").length == 2) {
        for (const msg of subpkgs.pkgs[name]) {
          const li = document.createElement("li");
          const a = document.createElement("a");
          a.href = "#" + msg;
          a.innerText = msg;
          a.style.fontSize = "0.90rem";
          a.style.fontWeight = 400;
          li.appendChild(a);
          li.style.whiteSpace = "nowrap";
          toc.getElementsByTagName("ul")[0].appendChild(li);
        }
        continue;
      }
      const li = document.createElement("li");
      const buttonTmpl = document.createElement("template");
      buttonTmpl.innerHTML =
        '<button class="btn btn-toggle align-items-center rounded collapsed" style="font-size:0.90rem" data-bs-toggle="collapse">' +
        name +
        "</button>";
      li.appendChild(buttonTmpl.content.firstChild);
      li.style.whiteSpace = "nowrap";
      toc.getElementsByTagName("ul")[0].appendChild(li);

      const ul = document.createElement("ul");
      li.classList.add("btn-toggle-nav");
      li.addEventListener("click", function () {
        if (ul.style.display == "none") {
          ul.style.display = "block";
        } else {
          ul.style.display = "none";
        }
      });
      const h = window.location.hash.replace("#", "");
      if (h && h.startsWith(name)) {
        ul.style.display = "block";
      } else {
        ul.style.display = "none";
      }
      li.appendChild(ul);
      for (const msg of subpkgs.pkgs[name]) {
        const li = document.createElement("li");
        msgText = msg.replace(name, "");
        const a = document.createElement("a");
        a.href = "#" + msg;
        a.innerText = msgText;
        a.style.fontSize = "0.90rem";
        li.appendChild(a);
        a.addEventListener("click", function (e) {
          e.stopPropagation();
          ul.style.display = "block";
        });
        ul.appendChild(li);
      }
    }
  }
};

document.addEventListener("DOMContentLoaded", async function () {
  setLang();
  addButtonHandler();
  markConfigDocLinkActive();
  for (const el of document.getElementsByClassName("protodoc")) {
    el.style.borderLeft = "3px solid #e76f51";
  }
  addTOC();
});
