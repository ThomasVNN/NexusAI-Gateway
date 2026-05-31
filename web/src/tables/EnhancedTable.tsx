import { ReactNode, useState } from 'react';
import {
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  MoreHorizontal,
  Edit,
  Trash2,
  Copy,
} from 'lucide-react';

export interface Column<T> {
  key: string;
  header: string;
  sortable?: boolean;
  width?: string;
  render?: (row: T) => ReactNode;
}

interface EnhancedTableProps<T extends { id: string | number }> {
  columns: Column<T>[];
  data: T[];
  onRowClick?: (row: T) => void;
  onEdit?: (row: T) => void;
  onDelete?: (row: T) => void;
  onDuplicate?: (row: T) => void;
  emptyMessage?: string;
  loading?: boolean;
  stickyHeader?: boolean;
  actions?: boolean;
}

type SortDirection = 'asc' | 'desc' | null;

export function EnhancedTable<T extends { id: string | number }>({
  columns,
  data,
  onRowClick,
  onEdit,
  onDelete,
  onDuplicate,
  emptyMessage = 'No data available',
  loading = false,
  stickyHeader = true,
  actions = true,
}: EnhancedTableProps<T>) {
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);
  const [openMenuId, setOpenMenuId] = useState<string | number | null>(null);

  const handleSort = (key: string) => {
    if (sortKey === key) {
      if (sortDirection === 'asc') {
        setSortDirection('desc');
      } else if (sortDirection === 'desc') {
        setSortDirection(null);
        setSortKey(null);
      }
    } else {
      setSortKey(key);
      setSortDirection('asc');
    }
  };

  const sortedData = [...data].sort((a, b) => {
    if (!sortKey || !sortDirection) return 0;
    const aVal = (a as Record<string, unknown>)[sortKey];
    const bVal = (b as Record<string, unknown>)[sortKey];
    if (aVal === bVal) return 0;
    if (aVal === null || aVal === undefined) return 1;
    if (bVal === null || bVal === undefined) return -1;
    const comparison = aVal < bVal ? -1 : 1;
    return sortDirection === 'asc' ? comparison : -comparison;
  });

  const getSortIcon = (key: string) => {
    if (sortKey !== key) {
      return <ChevronsUpDown className="w-4 h-4 text-text-muted" />;
    }
    return sortDirection === 'asc' ? (
      <ChevronUp className="w-4 h-4 text-accent-primary" />
    ) : (
      <ChevronDown className="w-4 h-4 text-accent-primary" />
    );
  };

  const toggleMenu = (id: string | number, e: React.MouseEvent) => {
    e.stopPropagation();
    setOpenMenuId(openMenuId === id ? null : id);
  };

  if (loading) {
    return (
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl overflow-hidden">
        <div className="animate-pulse">
          <div className="h-12 bg-bg-elevated border-b border-border-subtle" />
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-16 border-b border-border-subtle flex items-center px-4 gap-4">
              {columns.map((_, j) => (
                <div key={j} className="h-4 bg-bg-elevated rounded flex-1" />
              ))}
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-bg-tertiary border border-border-subtle rounded-xl overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className={`bg-bg-secondary ${stickyHeader ? 'sticky top-0 z-10' : ''}`}>
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={`px-4 py-3 text-left text-xs font-medium text-text-tertiary uppercase tracking-wider ${
                    column.sortable ? 'cursor-pointer hover:text-text-secondary select-none' : ''
                  }`}
                  style={{ width: column.width }}
                  onClick={() => column.sortable && handleSort(column.key)}
                >
                  <div className="flex items-center gap-2">
                    {column.header}
                    {column.sortable && getSortIcon(column.key)}
                  </div>
                </th>
              ))}
              {actions && <th className="px-4 py-3 w-12" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-border-subtle">
            {sortedData.length === 0 ? (
              <tr>
                <td colSpan={columns.length + (actions ? 1 : 0)} className="px-4 py-12 text-center text-text-muted">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              sortedData.map((row, rowIndex) => (
                <tr
                  key={row.id}
                  className={`hover:bg-bg-elevated/50 transition-colors ${
                    rowIndex % 2 === 1 ? 'bg-bg-secondary/30' : ''
                  } ${onRowClick ? 'cursor-pointer' : ''}`}
                  onClick={() => onRowClick?.(row)}
                >
                  {columns.map((column) => (
                    <td key={column.key} className="px-4 py-3 text-sm text-text-primary">
                      {column.render
                        ? column.render(row)
                        : String((row as Record<string, unknown>)[column.key] ?? '')}
                    </td>
                  ))}
                  {actions && (
                    <td className="px-4 py-3 relative">
                      <button
                        onClick={(e) => toggleMenu(row.id, e)}
                        className="p-1 rounded hover:bg-bg-elevated text-text-muted hover:text-text-secondary transition-colors"
                      >
                        <MoreHorizontal className="w-4 h-4" />
                      </button>
                      {openMenuId === row.id && (
                        <div className="absolute right-4 top-full mt-1 w-40 bg-bg-secondary border border-border-subtle rounded-lg shadow-lg py-1 z-20">
                          {onEdit && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                onEdit(row);
                                setOpenMenuId(null);
                              }}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-text-secondary hover:bg-bg-elevated transition-colors"
                            >
                              <Edit className="w-4 h-4" />
                              Edit
                            </button>
                          )}
                          {onDuplicate && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                onDuplicate(row);
                                setOpenMenuId(null);
                              }}
                              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-text-secondary hover:bg-bg-elevated transition-colors"
                            >
                              <Copy className="w-4 h-4" />
                              Duplicate
                            </button>
                          )}
                          {onDelete && (
                            <>
                              <div className="border-t border-border-subtle my-1" />
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  onDelete(row);
                                  setOpenMenuId(null);
                                }}
                                className="w-full flex items-center gap-2 px-4 py-2 text-sm text-error hover:bg-error/10 transition-colors"
                              >
                                <Trash2 className="w-4 h-4" />
                                Delete
                              </button>
                            </>
                          )}
                        </div>
                      )}
                    </td>
                  )}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
