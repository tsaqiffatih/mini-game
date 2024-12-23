import { Outlet } from "react-router-dom";
import ToggleTheme from "./ToggleTheme";


const MainLayout = () => {
  return (
    <div className="min-h-screen bg-base-100">
      {/* Tambahkan tombol toggle tema di sini */}
      <div className="absolute top-4 right-4">
        <ToggleTheme />
      </div>
      {/* Outlet akan menampilkan konten halaman */}
      <Outlet />
    </div>
  );
};

export default MainLayout;
