import Link from 'next/link';

export default function NotFound() {
    return (
      <div className="flex flex-col items-center justify-center h-screen">
        <h1 className="text-4xl font-bold">404 - Page Not Found</h1>
        <p className="text-lg mt-4">Sorry, the page you&apos;re looking for doesn&apos;t exist.</p>
        <Link href="/" passHref>
          <h2 className="mt-6 btn btn-primary btn-outline text-primary">Back To Home</h2>
        </Link>
      </div>
    );
  }
  