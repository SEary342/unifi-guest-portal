document.addEventListener("DOMContentLoaded", () => {
  const form = document.getElementById("login-form") as HTMLFormElement;
  const usernameInput = document.getElementById("username") as HTMLInputElement;
  const emailInput = document.getElementById("email") as HTMLInputElement;

  form?.addEventListener("submit", async (event) => {
    event.preventDefault();

    const username = usernameInput.value;
    const email = emailInput.value;
    const cacheId = window.cacheId;

    if (username && cacheId) {
      // Prepare the request body
      const requestBody = {
        username,
        email,
        cacheId,
      };

      try {
        const response = await fetch("/api/login", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(requestBody),
        });

        if (response.redirected) {
          // If redirected, follow the new location
          window.location.href = response.url;
        } else if (response.ok) {
          // Handle successful login if no redirect
          console.log('Login successful!');
          // Optionally, handle the success logic here
        } else {
          // Handle error if response is not successful (non-3xx, non-200)
          console.error('Login failed', response.statusText);
          // Optionally show an error message on the page
        }
      } catch (error) {
        console.error("Request failed", error);
      }

      // Optionally clear the form fields after submission
      usernameInput.value = '';
      emailInput.value = '';
    } else {
      alert("Please enter both a username and cacheId.");
    }
  });
});
