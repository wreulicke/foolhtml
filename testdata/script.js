console.log("Inlined script is running!");
document.addEventListener("DOMContentLoaded", function() {
    const p = document.createElement("p");
    p.textContent = "This paragraph was added by an inlined script.";
    p.style.fontWeight = "bold";
    document.body.appendChild(p);
});