document.addEventListener("DOMContentLoaded", () => {
  const form = document.getElementById("login-form") as HTMLFormElement;
  const usernameInput = document.getElementById("username") as HTMLInputElement;
  const emailInput = document.getElementById("email") as HTMLInputElement;

  form?.addEventListener("submit", (event) => {
    event.preventDefault();

    const username = usernameInput.value;
    const email = emailInput.value;

    if (username) {
      // If username is provided, handle login logic
      console.log("Logging in with", username);
      if (email) {
        console.log("Email:", email);  // Optional, only logs if email is provided
      }
      
      // Optionally clear the form fields after submission
      usernameInput.value = '';
      emailInput.value = '';
    } else {
      alert("Please enter your username.");
    }
  });
});
