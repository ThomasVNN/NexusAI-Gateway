import React from 'react';

export type InputSize = 'sm' | 'md' | 'lg';
export type InputVariant = 'default' | 'error';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  inputSize?: InputSize;
  variant?: InputVariant;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  errorMessage?: string;
}

const sizeStyles: Record<InputSize, string> = {
  sm: 'h-8 px-3 text-xs',
  md: 'h-10 px-3.5 text-sm',
  lg: 'h-12 px-4 text-base',
};

const variantStyles: Record<InputVariant, string> = {
  default: 'border-border-default focus:border-border-focus',
  error: 'border-error focus:border-error focus:ring-2 focus:ring-error/20',
};

export function Input({
  inputSize = 'md',
  variant = 'default',
  leftIcon,
  rightIcon,
  errorMessage,
  className = '',
  ...props
}: InputProps) {
  const hasError = variant === 'error' || errorMessage;

  return (
    <div className="w-full">
      <div
        className={`
          relative flex items-center
          bg-bg-secondary border rounded-lg
          transition-all duration-100
          ${sizeStyles[inputSize]}
          ${hasError ? variantStyles.error : variantStyles.default}
          ${className}
        `.trim().replace(/\s+/g, ' ')}
      >
        {leftIcon && (
          <span className="absolute left-3 flex items-center text-text-muted">
            {leftIcon}
          </span>
        )}
        <input
          className={`
            w-full h-full bg-transparent
            text-text-primary placeholder:text-text-muted
            focus:outline-none
            ${leftIcon ? 'pl-10' : ''}
            ${rightIcon ? 'pr-10' : ''}
          `.trim().replace(/\s+/g, ' ')}
          {...props}
        />
        {rightIcon && (
          <span className="absolute right-3 flex items-center text-text-muted">
            {rightIcon}
          </span>
        )}
      </div>
      {errorMessage && (
        <p className="mt-1 text-xs text-error">{errorMessage}</p>
      )}
    </div>
  );
}

export default Input;
