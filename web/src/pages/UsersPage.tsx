import { useState, useEffect } from 'react';
import {
  Plus,
  User,
  Shield,
  Mail,
  Key,
  MoreHorizontal,
  Edit,
  Trash2,
  ToggleLeft,
  ToggleRight,
} from 'lucide-react';
import { EnhancedTable, type Column } from '../tables/EnhancedTable';

interface User {
  id: string;
  name: string;
  email: string;
  role: 'admin' | 'member' | 'viewer';
  quota_daily: number;
  quota_used_today: number;
  is_active: boolean;
  created_at: string;
  last_active: string;
}

const roleColors: Record<string, string> = {
  admin: 'bg-accent-primary/10 text-accent-primary',
  member: 'bg-info/10 text-info',
  viewer: 'bg-text-muted/10 text-text-muted',
};

export function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    try {
      const res = await fetch('/api/admin/users');
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || []);
      }
    } catch {
      setUsers([
        {
          id: 'usr-1',
          name: 'Admin User',
          email: 'admin@nexusai.local',
          role: 'admin',
          quota_daily: 100000,
          quota_used_today: 24500,
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          last_active: '2024-05-31T10:30:00Z',
        },
        {
          id: 'usr-2',
          name: 'John Developer',
          email: 'john@company.com',
          role: 'member',
          quota_daily: 50000,
          quota_used_today: 12300,
          is_active: true,
          created_at: '2024-02-15T08:00:00Z',
          last_active: '2024-05-30T16:45:00Z',
        },
        {
          id: 'usr-3',
          name: 'Sarah Analyst',
          email: 'sarah@company.com',
          role: 'member',
          quota_daily: 25000,
          quota_used_today: 8900,
          is_active: true,
          created_at: '2024-03-01T12:00:00Z',
          last_active: '2024-05-31T09:15:00Z',
        },
        {
          id: 'usr-4',
          name: 'Guest Viewer',
          email: 'guest@example.com',
          role: 'viewer',
          quota_daily: 5000,
          quota_used_today: 1200,
          is_active: false,
          created_at: '2024-04-10T14:30:00Z',
          last_active: '2024-05-20T11:00:00Z',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const toggleUser = async (user: User) => {
    const updated = { ...user, is_active: !user.is_active };
    try {
      await fetch(`/api/admin/users/${user.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: updated.is_active }),
      });
      setUsers((prev) => prev.map((u) => (u.id === user.id ? updated : u)));
    } catch (error) {
      console.error('Failed to toggle user:', error);
    }
  };

  const deleteUser = async (user: User) => {
    if (!confirm(`Delete user "${user.name}"? This action cannot be undone.`)) return;
    try {
      await fetch(`/api/admin/users/${user.id}`, { method: 'DELETE' });
      setUsers((prev) => prev.filter((u) => u.id !== user.id));
    } catch (error) {
      console.error('Failed to delete user:', error);
    }
  };

  const columns: Column<User>[] = [
    {
      key: 'name',
      header: 'User',
      sortable: true,
      render: (row) => (
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-accent-primary/10 flex items-center justify-center">
            <User className="w-5 h-5 text-accent-primary" />
          </div>
          <div>
            <p className="font-medium text-text-primary">{row.name}</p>
            <p className="text-xs text-text-muted flex items-center gap-1">
              <Mail className="w-3 h-3" />
              {row.email}
            </p>
          </div>
        </div>
      ),
    },
    {
      key: 'role',
      header: 'Role',
      width: '120px',
      render: (row) => (
        <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium capitalize ${roleColors[row.role]}`}>
          <Shield className="w-3 h-3" />
          {row.role}
        </span>
      ),
    },
    {
      key: 'quota',
      header: 'Daily Quota',
      width: '180px',
      render: (row) => {
        const usagePercent = (row.quota_used_today / row.quota_daily) * 100;
        return (
          <div>
            <div className="flex items-center justify-between text-xs mb-1">
              <span className="text-text-secondary">{row.quota_used_today.toLocaleString()} / {row.quota_daily.toLocaleString()}</span>
              <span className="text-text-muted">{usagePercent.toFixed(0)}%</span>
            </div>
            <div className="w-32 h-1.5 bg-bg-elevated rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${
                  usagePercent > 90 ? 'bg-error' : usagePercent > 70 ? 'bg-warning' : 'bg-success'
                }`}
                style={{ width: `${Math.min(usagePercent, 100)}%` }}
              />
            </div>
          </div>
        );
      },
    },
    {
      key: 'is_active',
      header: 'Status',
      width: '120px',
      render: (row) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            toggleUser(row);
          }}
          className="flex items-center gap-2"
        >
          {row.is_active ? (
            <ToggleRight className="w-5 h-5 text-success" />
          ) : (
            <ToggleLeft className="w-5 h-5 text-text-muted" />
          )}
          <span className={`text-sm ${row.is_active ? 'text-success' : 'text-text-muted'}`}>
            {row.is_active ? 'Active' : 'Inactive'}
          </span>
        </button>
      ),
    },
    {
      key: 'last_active',
      header: 'Last Active',
      sortable: true,
      width: '140px',
      render: (row) => (
        <span className="text-sm text-text-secondary">
          {new Date(row.last_active).toLocaleDateString()}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Users</h1>
          <p className="text-sm text-text-tertiary mt-1">Manage team members and access permissions</p>
        </div>
        <button
          onClick={() => {
            setEditingUser(null);
            setShowModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
        >
          <Plus className="w-4 h-4" />
          Add User
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
        {[
          { label: 'Total Users', value: users.length, icon: User },
          { label: 'Active', value: users.filter((u) => u.is_active).length, icon: ToggleRight, color: 'text-success' },
          { label: 'Admins', value: users.filter((u) => u.role === 'admin').length, icon: Shield, color: 'text-accent-primary' },
          { label: 'Members', value: users.filter((u) => u.role === 'member').length, icon: Key, color: 'text-info' },
        ].map((stat, i) => (
          <div key={i} className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-text-tertiary">{stat.label}</p>
                <p className={`text-2xl font-bold text-text-primary mt-1`}>{stat.value}</p>
              </div>
              <stat.icon className={`w-8 h-8 ${stat.color || 'text-text-muted'}`} />
            </div>
          </div>
        ))}
      </div>

      {/* Users Table */}
      <EnhancedTable
        columns={columns}
        data={users}
        loading={loading}
        onEdit={(user) => {
          setEditingUser(user);
          setShowModal(true);
        }}
        onDelete={deleteUser}
        emptyMessage="No users found. Invite your first team member."
      />

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-bg-secondary border border-border-subtle rounded-xl p-6 w-full max-w-lg">
            <h2 className="text-lg font-semibold text-text-primary mb-4">
              {editingUser ? 'Edit User' : 'Add New User'}
            </h2>
            <form className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Name</label>
                <input
                  type="text"
                  defaultValue={editingUser?.name || ''}
                  placeholder="Full name"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Email</label>
                <input
                  type="email"
                  defaultValue={editingUser?.email || ''}
                  placeholder="email@example.com"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Role</label>
                <select
                  defaultValue={editingUser?.role || 'member'}
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                >
                  <option value="admin">Admin - Full access</option>
                  <option value="member">Member - Standard access</option>
                  <option value="viewer">Viewer - Read-only</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Daily Quota</label>
                <input
                  type="number"
                  defaultValue={editingUser?.quota_daily || 10000}
                  min={0}
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                />
              </div>
              <div className="flex justify-end gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-text-secondary hover:text-text-primary transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  onClick={(e) => {
                    e.preventDefault();
                    setShowModal(false);
                    fetchUsers();
                  }}
                  className="px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
                >
                  {editingUser ? 'Save Changes' : 'Add User'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
