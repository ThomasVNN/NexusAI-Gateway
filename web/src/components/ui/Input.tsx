import React, { forwardRef } from 'react';

// ============================================================================
// Type Definitions
// ============================================================================

export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  label?: string;
  helperText?: string;
  error?: string;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  size?: 'sm' | 'md' | 'lg';
}

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  helperText?: string;
  error?: string;
}

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  helperText?: string;
  error?: string;
  options: { value: string; label: string; disabled?: boolean }[];
  placeholder?: string;
}

export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: string;
  description?: string;
}

export interface SwitchProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: string;
  description?: string;
  size?: 'sm' | 'md';
}

// ============================================================================
// Size Variants
// ============================================================================

const inputSizes = {
  sm: 'h-8 px-3 text-xs',
  md: 'h-10 px-3 text-sm',
  lg: 'h-12 px-4 text-base',
};

const textareaSizes = {
  sm: 'min-h-20 text-xs',
  md: 'min-h-24 text-sm',
  lg: 'min-h-32 text-base',
};

// ============================================================================
// Base Input Component
// ============================================================================

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, helperText, error, leftIcon, rightIcon, size = 'md', className = '', id, ...props }, ref) => {
    const inputId = id || `input-${Math.random().toString(36).substr(2, 9)}`;
    const hasError = Boolean(error);

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={inputId}
            className="block text-sm font-medium text-text-secondary mb-1.5"
          >
            {label}
          </label>
        )}
        <div className="relative">
          {leftIcon && (
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none">
              {leftIcon}
            </div>
          )}
          <input
            ref={ref}
            id={inputId}
            className={`
              w-full
              bg-bg-secondary border rounded-lg
              text-text-primary placeholder-text-muted
              transition-all duration-100
              focus:outline-none focus:border-border-focus focus:ring-2 focus:ring-accent-primary/20
              disabled:bg-bg-muted disabled:cursor-not-allowed disabled:text-text-muted
              ${hasError ? 'border-error focus:border-error focus:ring-error/20' : 'border-border-default'}
              ${leftIcon ? 'pl-10' : ''}
              ${rightIcon ? 'pr-10' : ''}
              ${inputSizes[size]}
              ${className}
            `.trim().replace(/\s+/g, ' ')}
            aria-invalid={hasError}
            aria-describedby={error ? `${inputId}-error` : helperText ? `${inputId}-helper` : undefined}
            {...props}
          />
          {rightIcon && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none">
              {rightIcon}
            </div>
          )}
        </div>
        {error && (
          <p id={`${inputId}-error`} className="mt-1.5 text-xs text-error" role="alert">
            {error}
          </p>
        )}
        {!error && helperText && (
          <p id={`${inputId}-helper`} className="mt-1.5 text-xs text-text-tertiary">
            {helperText}
          </p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';

// ============================================================================
// Textarea Component
// ============================================================================

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ label, helperText, error, size = 'md', className = '', id, ...props }, ref) => {
    const textareaId = id || `textarea-${Math.random().toString(36).substr(2, 9)}`;
    const hasError = Boolean(error);

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={textareaId}
            className="block text-sm font-medium text-text-secondary mb-1.5"
          >
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          id={textareaId}
          className={`
            w-full px-3 py-2
            bg-bg-secondary border rounded-lg
            text-text-primary placeholder-text-muted
            resize-y transition-all duration-100
            focus:outline-none focus:border-border-focus focus:ring-2 focus:ring-accent-primary/20
            disabled:bg-bg-muted disabled:cursor-not-allowed disabled:text-text-muted
            ${hasError ? 'border-error focus:border-error focus:ring-error/20' : 'border-border-default'}
            ${textareaSizes[size]}
            ${className}
          `.trim().replace(/\s+/g, ' ')}
          aria-invalid={hasError}
          aria-describedby={error ? `${textareaId}-error` : helperText ? `${textareaId}-helper` : undefined}
          {...props}
        />
        {error && (
          <p id={`${textareaId}-error`} className="mt-1.5 text-xs text-error" role="alert">
            {error}
          </p>
        )}
        {!error && helperText && (
          <p id={`${textareaId}-helper`} className="mt-1.5 text-xs text-text-tertiary">
            {helperText}
          </p>
        )}
      </div>
    );
  }
);

Textarea.displayName = 'Textarea';

// ============================================================================
// Select Component
// ============================================================================

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, helperText, error, options, placeholder, size = 'md', className = '', id, ...props }, ref) => {
    const selectId = id || `select-${Math.random().toString(36).substr(2, 9)}`;
    const hasError = Boolean(error);

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={selectId}
            className="block text-sm font-medium text-text-secondary mb-1.5"
          >
            {label}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={selectId}
            className={`
              w-full appearance-none
              bg-bg-secondary border rounded-lg
              text-text-primary
              transition-all duration-100
              focus:outline-none focus:border-border-focus focus:ring-2 focus:ring-accent-primary/20
              disabled:bg-bg-muted disabled:cursor-not-allowed disabled:text-text-muted
              ${hasError ? 'border-error focus:border-error focus:ring-error/20' : 'border-border-default'}
              ${inputSizes[size]}
              pr-10
              ${className}
            `.trim().replace(/\s+/g, ' ')}
            aria-invalid={hasError}
            aria-describedby={error ? `${selectId}-error` : helperText ? `${selectId}-helper` : undefined}
            {...props}
          >
            {placeholder && (
              <option value="" disabled>
                {placeholder}
              </option>
            )}
            {options.map((option) => (
              <option key={option.value} value={option.value} disabled={option.disabled}>
                {option.label}
              </option>
            ))}
          </select>
          <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </div>
        </div>
        {error && (
          <p id={`${selectId}-error`} className="mt-1.5 text-xs text-error" role="alert">
            {error}
          </p>
        )}
        {!error && helperText && (
          <p id={`${selectId}-helper`} className="mt-1.5 text-xs text-text-tertiary">
            {helperText}
          </p>
        )}
      </div>
    );
  }
);

Select.displayName = 'Select';

// ============================================================================
// Checkbox Component
// ============================================================================

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
  ({ label, description, className = '', id, ...props }, ref) => {
    const checkboxId = id || `checkbox-${Math.random().toString(36).substr(2, 9)}`;

    return (
      <div className={`flex items-start gap-3 ${className}`}>
        <div className="flex items-center h-5 mt-0.5">
          <input
            ref={ref}
            id={checkboxId}
            type="checkbox"
            className="
              w-4 h-4
              bg-bg-secondary border border-border-default rounded
              text-accent-primary
              cursor-pointer
              transition-all duration-100
              focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus focus-visible:ring-offset-2 focus-visible:ring-offset-bg-primary
              checked:bg-accent-primary checked:border-accent-primary
              disabled:bg-bg-muted disabled:cursor-not-allowed
            "
            {...props}
          />
        </div>
        {(label || description) && (
          <div className="flex-1">
            {label && (
              <label htmlFor={checkboxId} className="text-sm font-medium text-text-primary cursor-pointer">
                {label}
              </label>
            )}
            {description && (
              <p className="text-xs text-text-tertiary mt-0.5">{description}</p>
            )}
          </div>
        )}
      </div>
    );
  }
);

Checkbox.displayName = 'Checkbox';

// ============================================================================
// Switch Component
// ============================================================================

export const Switch = forwardRef<HTMLInputElement, SwitchProps>(
  ({ label, description, size = 'md', className = '', id, ...props }, ref) => {
    const switchId = id || `switch-${Math.random().toString(36).substr(2, 9)}`;
    const sizes = {
      sm: { track: 'w-8 h-4', thumb: 'w-3 h-3', translate: 'translate-x-4', padding: 'py-1' },
      md: { track: 'w-11 h-6', thumb: 'w-5 h-5', translate: 'translate-x-5', padding: 'py-0.5' },
    };
    const sizeConfig = sizes[size];

    return (
      <div className={`flex items-center justify-between gap-4 ${className}`}>
        {(label || description) && (
          <div className="flex-1">
            {label && (
              <label htmlFor={switchId} className="text-sm font-medium text-text-primary cursor-pointer">
                {label}
              </label>
            )}
            {description && (
              <p className="text-xs text-text-tertiary mt-0.5">{description}</p>
            )}
          </div>
        )}
        <div className={`relative ${sizeConfig.padding}`}>
          <input
            ref={ref}
            id={switchId}
            type="switch"
            role="switch"
            className="
              sr-only peer
              checked:bg-accent-primary
              focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus focus-visible:ring-offset-2 focus-visible:ring-offset-bg-primary
              disabled:bg-bg-muted disabled:cursor-not-allowed
            "
            {...props}
          />
          <div className={`
            ${sizeConfig.track}
            bg-bg-muted rounded-full
            cursor-pointer
            transition-colors duration-200
            peer-checked:bg-accent-primary
            peer-focus-visible:ring-2 peer-focus-visible:ring-border-focus peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-bg-primary
            peer-disabled:cursor-not-allowed peer-disabled:opacity-50
          `}>
            <div className={`
              ${sizeConfig.thumb}
              absolute top-0.5 left-0.5
              bg-white rounded-full shadow
              transition-transform duration-200
              peer-checked:${sizeConfig.translate}
            `} />
          </div>
        </div>
      </div>
    );
  }
);

Switch.displayName = 'Switch';

export default Input;
