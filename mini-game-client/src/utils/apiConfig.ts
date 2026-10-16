export const getHttpBackendUrl = (): string => {
  const envUrl = process.env.NEXT_PUBLIC_HTTP_BACKEND_URL;
  if (envUrl && envUrl.trim() !== "") {
    return envUrl.trim().replace(/\/+$/, "");
  }
  return "";
};

export const getWsBackendUrl = (): string => {
  const envUrl = process.env.NEXT_PUBLIC_WS_BACKEND_URL;
  if (envUrl && envUrl.trim() !== "") {
    return envUrl.trim().replace(/\/+$/, "");
  }
  if (typeof window !== "undefined") {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${window.location.host}`;
  }
  return "";
};
