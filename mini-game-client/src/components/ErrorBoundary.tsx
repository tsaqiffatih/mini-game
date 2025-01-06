import React, { Component, ReactNode } from "react";

interface ErrorBoundaryProps {
  children: ReactNode; // Komponen anak-anak yang dibungkus oleh Error Boundary
  fallback?: ReactNode; // UI fallback opsional yang akan ditampilkan saat error terjadi
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
    };
  }

  // Lifecycle method untuk menangkap error
  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  // Optional: Kirim error ke logging service
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    console.error("Error caught by Error Boundary:", error, errorInfo);
    // Contoh: kirim log ke server
    // sendErrorLogToServer(error, errorInfo);
  }

  // Reset state error saat pengguna ingin mencoba kembali
  handleReset = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    const { hasError, error } = this.state;
    const { children, fallback } = this.props;

    if (hasError) {
      return (
        <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100 p-4">
          <div className="bg-white shadow-md rounded-lg p-6 max-w-md text-center">
            <h1 className="text-2xl font-bold text-red-600 mb-4">Oops! Something went wrong.</h1>
            {fallback || (
              <>
                <p className="text-gray-700 mb-4">{error?.message}</p>
                <button
                  onClick={this.handleReset}
                  className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-700 transition"
                >
                  Try Again
                </button>
              </>
            )}
          </div>
        </div>
      );
    }

    return children;
  }
}