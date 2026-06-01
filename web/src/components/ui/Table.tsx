import React from 'react';

export interface TableProps extends React.HTMLAttributes<HTMLTableElement> {
  children: React.ReactNode;
}

export interface TableHeaderProps extends React.HTMLAttributes<HTMLTableSectionElement> {
  children: React.ReactNode;
}

export interface TableBodyProps extends React.HTMLAttributes<HTMLTableSectionElement> {
  children: React.ReactNode;
}

export interface TableRowProps extends React.HTMLAttributes<HTMLTableRowElement> {
  children: React.ReactNode;
  isSelected?: boolean;
}

export interface TableHeadProps extends React.ThHTMLAttributes<HTMLTableCellElement> {
  children: React.ReactNode;
}

export interface TableCellProps extends React.TdHTMLAttributes<HTMLTableCellElement> {
  children: React.ReactNode;
}

export function Table({ children, className = '', ...props }: TableProps) {
  return (
    <div className="w-full overflow-x-auto">
      <table
        className={`w-full border-collapse ${className}`}
        {...props}
      >
        {children}
      </table>
    </div>
  );
}

export function TableHeader({ children, className = '', ...props }: TableHeaderProps) {
  return (
    <thead
      className={`bg-bg-elevated border-b border-border-subtle ${className}`}
      {...props}
    >
      {children}
    </thead>
  );
}

export function TableBody({ children, className = '', ...props }: TableBodyProps) {
  return (
    <tbody
      className={`divide-y divide-border-subtle ${className}`}
      {...props}
    >
      {children}
    </tbody>
  );
}

export function TableRow({ children, isSelected, className = '', ...props }: TableRowProps) {
  return (
    <tr
      className={`
        transition-colors duration-100
        hover:bg-bg-elevated/50
        ${isSelected ? 'bg-accent-primary/5' : ''}
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      {...props}
    >
      {children}
    </tr>
  );
}

export function TableHead({ children, className = '', ...props }: TableHeadProps) {
  return (
    <th
      className={`
        px-4 py-3 text-left text-xs font-medium text-text-tertiary uppercase tracking-wider
        ${className}
      `.trim()}
      {...props}
    >
      {children}
    </th>
  );
}

export function TableCell({ children, className = '', ...props }: TableCellProps) {
  return (
    <td
      className={`px-4 py-3 text-sm text-text-primary ${className}`}
      {...props}
    >
      {children}
    </td>
  );
}

export default Table;
