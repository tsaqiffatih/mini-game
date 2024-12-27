export default function playMoveSound(): void {
    const moveSound = new Audio("./sounds/Move.mp3");
    const playMove = moveSound.play();
    if (playMove !== undefined) {
        playMove
            .then(() => {})
            .catch((error: Error) => {
                console.log("Failed to load move sound", error);
            });
    }
}