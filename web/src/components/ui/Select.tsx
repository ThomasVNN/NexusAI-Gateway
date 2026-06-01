import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown, Check } from 'lucide-react';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps {
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  children: React.ReactNode;
}

export interface SelectTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  placeholder?: string;
  children?: React.ReactNode;
}

export interface SelectValueProps {
  placeholder?: string;
  value?: string;
}

export interface SelectContentProps {
  children: React.ReactNode;
  className?: string;
}

export interface SelectItemProps extends React.OptionHTMLAttributes<HTMLOptionElement> {
  value: string;
  children: React.ReactNode;
}

export function Select({
  value,
  defaultValue,
  onValueChange,
  placeholder = 'Select...',
  disabled = false,
  className = '',
  children,
}: SelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedValue, setSelectedValue] = useState(value || defaultValue || '');
  const selectRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (selectRef.current && !selectRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = (newValue: string) => {
    setSelectedValue(newValue);
    onValueChange?.(newValue);
    setIsOpen(false);
  };

  return (
    <div ref={selectRef} className={`relative ${className}`}>
      <SelectTrigger
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className="w-full"
      >
        <SelectValue value={selectedValue} placeholder={placeholder} />
        <ChevronDown className={`w-4 h-4 text-text-muted transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </SelectTrigger>

      {isOpen && (
        <SelectContent className="absolute z-50 mt-1 w-full min-w-[8rem]">
          {children}
        </SelectContent>
      )}
    </div>
  );
}

export function SelectTrigger({
  placeholder,
  children,
  className = '',
  ...props
}: SelectTriggerProps) {
  return (
    <button
      type="button"
      className={`
        flex items-center justify-between
        w-full px-3.5 py-2
        bg-bg-secondary border border-border-default
        rounded-lg text-sm
        transition-all duration-100
        focus:outline-none focus:border-border-focus focus:ring-2 focus:ring-accent-primary/20
        disabled:opacity-50 disabled:cursor-not-allowed
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      {...props}
    >
      {children || <span className="text-text-muted">{placeholder}</span>}
    </button>
  );
}

export function SelectValue({ placeholder, value }: SelectValueProps) {
  if (!value) {
    return <span className="text-text-muted">{placeholder}</span>;
  }
  return <span className="text-text-primary">{value}</span>;
}

export function SelectContent({ children, className = '' }: SelectContentProps) {
  return (
    <div
      className={`
        overflow-hidden
        bg-bg-secondary border border-border-default
        rounded-lg shadow-lg
        animate-scale-in
        ${className}
      `.trim().replace(/\s+/g, ' ')}
    >
      {children}
    </div>
  );
}

export function SelectItem({ value, children, className = '', ...props }: SelectItemProps) {
  return (
    <option
      value={value}
      className={`
        px-3 py-2 text-sm text-text-primary
        cursor-pointer
        hover:bg-bg-elevated
        focus:bg-bg-elevated
        focus:outline-none
        ${className}
      `.trim()}
      {...props}
    >
      {children}
    </option>
  );
}

// Helper component for rendering options
export function SelectOptions({
  options,
  value,
  onChange,
}: {
  options: SelectOption[];
  value?: string;
  onChange?: (value: string) => void;
}) {
  return (
    <SelectContent>
      {options.map((option) => (
        <SelectItem
          key={option.value}
          value={option.value}
          disabled={option.disabled}
        >
          <div
            className="flex items-center justify-between"
            onClick={() => onChange?.(option.value)}
          >
            {option.label}
            {value === option.value && <Check className="w-4 h-4 text-accent-primary" />}
          </div>
        </SelectItem>
      ))}
    </SelectContent>
  );
}

export default Select;
