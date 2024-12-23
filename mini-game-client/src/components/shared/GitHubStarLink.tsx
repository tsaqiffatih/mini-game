export default function GitHubStarLink() {
  const href = "https://github.com/";

  return (
    <a
      href={href}
      target="_blank"
      className="inline-flex bottom-0 bg-gray-100 border border-primary justify-between items-center p-1 pr-4 mb-10 text-sm rounded-full hover:bg-gray-200"
      role="alert"
    >
      <span className="text-xs bg-primary text-base-100 rounded-full sm:px-4 px-2.5 py-1.5 mr-3">
        Repository
      </span>
      <span className="sm:text-sm text-xs font-medium">
        Support the project! Leave a star
      </span>
      <span className="ml-2">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          height="24"
          viewBox="0 0 24 24"
          width="24"
        >
          <path
            d="M11.59 7.41L16.17 12l-4.58 4.59L12 18l6-6-6-6-1.41 1.41z"
            fill="currentColor"
          />
        </svg>
      </span>
    </a>
  );
}
