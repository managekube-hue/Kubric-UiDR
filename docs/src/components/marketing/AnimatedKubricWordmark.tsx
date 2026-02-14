import React from 'react';
import {motion} from 'framer-motion';

const letters = 'KUBRIC'.split('');

export default function AnimatedKubricWordmark(): JSX.Element {
  return (
    <div className="kubricWordmarkWrap" aria-label="Kubric animated wordmark">
      <motion.div
        className="kubricWordmarkShutter"
        initial={{clipPath: 'inset(0 100% 0 0)'}}
        animate={{clipPath: 'inset(0 0% 0 0)'}}
        transition={{duration: 1.2, ease: 'easeOut'}}
      >
        {letters.map((letter, index) => (
          <motion.span
            className="kubricWordmarkLetter"
            key={`${letter}-${index}`}
            initial={{y: 0}}
            animate={{y: [0, -5, 0]}}
            transition={{
              duration: 2.2,
              ease: 'easeInOut',
              repeat: Infinity,
              delay: index * 0.08,
            }}
          >
            {letter}
          </motion.span>
        ))}
      </motion.div>
      <motion.div
        className="kubricWordmarkCursor"
        animate={{opacity: [0, 1, 0]}}
        transition={{duration: 0.9, repeat: Infinity}}
        aria-hidden="true"
      >
        ▌
      </motion.div>
    </div>
  );
}
