import Swal, { SweetAlertOptions } from "sweetalert2";


const getThemeClass = (): SweetAlertOptions["customClass"] => {
  const theme = localStorage.getItem("theme"); 
  return theme === "dark" ? { popup: "swal-theme-dark" } : { popup: "swal-theme-light" };
};

export const showAlert = (options: SweetAlertOptions) => {
  return Swal.fire({
    customClass: getThemeClass(),
    ...options,
  });
};

export const showSuccessAlert = (message: string) => {
  return showAlert({
    title: "Success",
    text: message,
    icon: "success",
  });
};

export const showErrorAlert = (message: string) => {
  return showAlert({
    title: "Uppsss...",
    text: message,
    icon: "error",
  });
};
