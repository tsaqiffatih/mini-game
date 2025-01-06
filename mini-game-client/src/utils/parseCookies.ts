export function parseCookies(cookieHeader: string) {
  return cookieHeader
    .split("; ")
    .reduce((acc: Record<string, string>, cookie) => {
      const [key, value] = cookie.split("=");
      acc[key] = decodeURIComponent(value);
      return acc;
    }, {});
}
