import { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
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

export const handleLeaveGameAlert = (router: AppRouterInstance) => {
  showAlert({
    title: "Leave Game?",
    text: "Are you sure you want to leave the game? Your progress will be lost.",
    icon: "warning",
    showCancelButton: true,
    confirmButtonText: "Yes, leave",
    cancelButtonText: "No, stay",
  }).then((result) => {
    if (result.isConfirmed) {
      localStorage.removeItem("roomId");
      localStorage.removeItem("playerMark");
      router.push("/");
    } else {
      window.history.pushState(null, "", window.location.href);
    }
  });
};
