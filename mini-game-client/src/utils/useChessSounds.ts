"use client";

import { useCallback, useEffect, useRef, useState } from "react";

export const useChessSounds = () => {
  const [audioUnlocked, setAudioUnlocked] = useState(false);

  const soundsRef = useRef({
    moveSelf: new Audio("/sounds/move-self.mp3"),
    moveOpponent: new Audio("/sounds/move-opponent.mp3"),
    capture: new Audio("/sounds/capture.mp3"),
    castle: new Audio("/sounds/castle.mp3"),
    check: new Audio("/sounds/check.mp3"),
    gameEnd: new Audio("/sounds/game-end.mp3"),
    illegal: new Audio("/sounds/illegal.mp3"),
    promote: new Audio("/sounds/promote.mp3"),
  });

  const play = useCallback(
    async (audio: HTMLAudioElement) => {
      if (!audioUnlocked) return;

      try {
        audio.currentTime = 0;
        await audio.play();
      } catch {}
    },
    [audioUnlocked],
  );

  const playMoveSelf = useCallback(() => {
    play(soundsRef.current.moveSelf);
  }, [play]);

  const playMoveOpponent = useCallback(() => {
    play(soundsRef.current.moveOpponent);
  }, [play]);

  const playCapture = useCallback(() => {
    play(soundsRef.current.capture);
  }, [play]);

  const playCastle = useCallback(() => {
    play(soundsRef.current.castle);
  }, [play]);

  const playCheck = useCallback(() => {
    play(soundsRef.current.check);
  }, [play]);

  const playGameEnd = useCallback(() => {
    play(soundsRef.current.gameEnd);
  }, [play]);

  const playIllegal = useCallback(() => {
    play(soundsRef.current.illegal);
  }, [play]);

  const playPromote = useCallback(() => {
    play(soundsRef.current.promote);
  }, [play]);

  useEffect(() => {
    const unlockAudio = () => {
      setAudioUnlocked(true);

      window.removeEventListener("pointerdown", unlockAudio);
    };

    window.addEventListener("pointerdown", unlockAudio);

    return () => {
      window.removeEventListener("pointerdown", unlockAudio);
    };
  }, []);

  return {
    playMoveSelf,
    playMoveOpponent,
    playCapture,
    playCastle,
    playCheck,
    playGameEnd,
    playIllegal,
    playPromote,
  };
};
