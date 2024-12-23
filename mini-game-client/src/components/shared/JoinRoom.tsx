import { useState } from "react";
import Modal from "./Modal";
import { joinRoom } from "../../api/gameApi";
import { showErrorAlert, showSuccessAlert } from "../../utils/alerthelper";
import { ApiError } from "../../interface";

interface JoinRoomProps {
  onJoinRoom: () => void;
}

export default function JoinRoom({ onJoinRoom }: JoinRoomProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    roomId: "",
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  const handleForm = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    console.log(formData);
    setIsModalOpen(false);

    const playerId = localStorage.getItem("playerId");

    if (!playerId) {
      await showErrorAlert("Player ID is missing. Please register first.");
      return;
    }

    try {
      const response = await joinRoom(formData.roomId, playerId);
      if (!response.success) {
        throw new Error(response.data.message);
      }

      console.log(response);

      localStorage.setItem("playerMark", response.data.player_mark);
      localStorage.setItem("roomId", response.data.room.room_id);
      await showSuccessAlert(
        `Success Join Room ${formData.roomId}. Enjoy the game!`
      );
      
      
      onJoinRoom();
    } catch (err) {
      const error = err as ApiError;
      if (error.response?.data) {
        await showErrorAlert(
          error.response.data.message || "Unexpected error occurred."
        );
      } else {
        await showErrorAlert("Error joining room.");
      }
    }
  };

  return (
    <div>
      <button
        className="btn btn-outline text-xl font-extrabold"
        onClick={() => setIsModalOpen(true)}
      >
        Join Room
      </button>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Join To Room"
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
              Join
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
