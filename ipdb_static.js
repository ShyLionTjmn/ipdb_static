function elmToggle(elm_id) {
  let e = document.getElementById(elm_id);
  if(e === null) return;
  let current_state = e.style.display;
  if(current_state === "none") {
    e.style.display = "block";
  } else {
    e.style.display = "none";
  };
};

function expandAll() {
  let list = document.querySelectorAll(".cont_data");
  for(let i = 0; i < list.length; i++) {
    list[i].style.display = "block";
  };
};

function collapseAll() {
  let list = document.querySelectorAll(".cont_data");
  for(let i = 0; i < list.length; i++) {
    list[i].style.display = "none";
  };
};
