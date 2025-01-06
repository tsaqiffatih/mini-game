"use client";

import { useState } from "react";
import Modal from "./Modal";
import axios, { AxiosError } from "axios";
import { showErrorAlert, showSuccessAlert } from "@/utils/alerthelper";
import { useRouter } from "next/navigation";

interface RegisterUserProps {
  onRegister: () => void;
}

const backendUrl = process.env.NEXT_PUBLIC_HTTP_BACKEND_URL;

export default function RegisterUser({ onRegister }: RegisterUserProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    userName: "",
  });

  const router = useRouter();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  const handleForm = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsModalOpen(false);

    try {
      const { data } = await axios.post(`${backendUrl}/create/user`, {
        player_id: formData.userName,
      });

      if (!data.success) {
        throw new Error(data.data.message);
      }

      await showSuccessAlert("Success Create Player");
      
      localStorage.setItem("playerId", data.data.player_id);

      onRegister()

      router.push("/");
    } catch (err) {
      const error = err as AxiosError<{ message: string }>;

      if (error.response?.data) {
        await showErrorAlert(
          error.response.data.message || "Unexpected error occurred."
        );
      } else {
        await showErrorAlert("Error registering player.");
      }
    }
  };

  return (
    <div>
      <button
        className="btn btn-primary text-xl font-extrabold"
        onClick={() => setIsModalOpen(true)}
      >
        <span>
          <svg
            fill="currentColor"
            width="20px"
            height="20px"
            viewBox="0 0 16 16"
            version="1.1"
            xmlns="http://www.w3.org/2000/svg"
          >
            <rect width="16" height="16" id="icon-bound" fill="none" />
            <path d="M14,14l0,-12l-6,0l0,-2l8,0l0,16l-8,0l0,-2l6,0Zm-6.998,-0.998l4.998,-5.002l-5,-5l-1.416,1.416l2.588,2.584l-8.172,0l0,2l8.172,0l-2.586,2.586l1.416,1.416Z" />
          </svg>
        </span>
        Register
      </button>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Sign Up"
      >
        <form className="space-y-4" onSubmit={handleForm}>
          <div>
            <label
              htmlFor="userName"
              className="mb-2 text-base-content text-lg"
            >
              Username
            </label>
            <input
              className="input input-bordered w-full"
              type="text"
              placeholder="Input your Username"
              required
              name="userName"
              value={formData.userName}
              onChange={handleChange}
            />
          </div>
          <div className="modal-action justify-between">
            <button
              type="button"
              className="btn btn-md btn-outline"
              onClick={() => setIsModalOpen(false)}
            >
              Close
            </button>
            <button className="btn btn-md btn-primary" type="submit">
              Register
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
