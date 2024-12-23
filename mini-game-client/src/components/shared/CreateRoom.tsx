import { useState } from "react";
import Modal from "./Modal";
import { createRoom } from "../../api/gameApi";
import { ApiError } from "../../interface";
import { showErrorAlert, showSuccessAlert } from "../../utils/alerthelper";

interface CreateRoomProps {
  onCreateRoom: () => void;
  gameType: string;
}

export default function CreateRoom({ onCreateRoom, gameType }: CreateRoomProps) {
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false);
  const [formData, setFormData] = useState<{ roomId: string }>({
    roomId: "",
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  const handleForm = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    console.log(formData);
    const playerID = localStorage.getItem("playerId") as string;
    onCreateRoom();
    setIsModalOpen(false);
    try {
      const response = await createRoom(playerID,formData.roomId);
      if (!response.success) {
        throw new Error(response.data.message);
      }

      await showSuccessAlert(`Success Create Room ${formData.roomId}. Now Join Room`);
    } catch (err) {
      const error = err as ApiError;
      if (error.response?.data) {
        await showErrorAlert(
          error.response.data.message || "Unexpected error occurred."
        );
      } else {
        await showErrorAlert("Error creating room.");
      }
    }
  };

  return (
    <div>
      <button
        className="btn btn-outline text-xl font-extrabold"
        onClick={() => setIsModalOpen(true)}
      >
        Create Room
      </button>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Create Your Own Room"
      >
        <form className="space-y-4" onSubmit={handleForm}>
          <div className="space-y-2">
            <label htmlFor="room_id" className="mb-2 text-base-content text-lg">
              Room ID
            </label>
            <input
              className="input input-bordered w-full"
              type="text"
              placeholder="Input Room ID"
              required
              name="roomId"
              value={formData.roomId}
              onChange={handleChange}
            />
          </div>
          <div className="modal-action justify-between">
            <button
              className="btn btn-outline"
              type="button"
              onClick={() => setIsModalOpen(false)}
            >
              Close
            </button>
            <button className="btn btn-primary" type="submit">
              Create
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
