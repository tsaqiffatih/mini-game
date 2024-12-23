import { Link } from "react-router-dom";

const NotFoundPage = () => {
  return (
    <div style={{ textAlign: "center", marginTop: "50px" }}>
      <h1>404 - Page Not Found</h1>
      <p>The page you are looking for doesnt exist or has been moved.</p>
      <Link to="/" style={{ textDecoration: "none", color: "blue" }}>
        Go Back to Home
      </Link>
    </div>
  );
};

export default NotFoundPage;
