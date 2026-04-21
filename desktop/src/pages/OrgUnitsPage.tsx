import React, { useEffect, useState } from 'react';
import {
  TreeView,
  TreeItem,
} from '@mui/lab';
import {
  Button,
  TextField,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  CircularProgress,
} from '@mui/material';

interface OrgUnit {
  id: number;
  name: string;
  parentId: number | null;
  path: string;
  children?: OrgUnit[];
}

export default function OrgUnitsPage() {
  const [units, setUnits] = useState<OrgUnit[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [newName, setNewName] = useState('');
  const [parentId, setParentId] = useState<number | ''>('');

  useEffect(() => {
    async function fetchUnits() {
      try {
        setLoading(true);
        const res = await fetch('/api/v1/sukumad/org_units');
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setUnits(json.items || json);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchUnits();
  }, []);

  function buildTree(list: OrgUnit[]): OrgUnit[] {
    const map: { [key: number]: OrgUnit & { children: OrgUnit[] } } = {};
    const roots: OrgUnit[] = [];
    list.forEach((u) => {
      map[u.id] = { ...u, children: [] };
    });
    list.forEach((u) => {
      if (u.parentId && map[u.parentId]) {
        map[u.parentId].children.push(map[u.id]);
      } else {
        roots.push(map[u.id]);
      }
    });
    return roots;
  }

  const treeData = buildTree(units);

  const handleOpen = () => {
    setNewName('');
    setParentId('');
    setOpenDialog(true);
  };
  const handleClose = () => setOpenDialog(false);
  const handleCreate = async () => {
    try {
      const payload = {
        name: newName,
        parentId: parentId === '' ? null : parentId,
      };
      const res = await fetch('/api/v1/sukumad/org_units', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      handleClose();
      const json = await res.json();
      setUnits((prev) => [...prev, json]);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const renderTree = (nodes: OrgUnit[]) =>
    nodes.map((n) => (
      <TreeItem key={n.id} nodeId={String(n.id)} label={n.name}>
        {n.children && n.children.length > 0 ? renderTree(n.children) : null}
      </TreeItem>
    ));

  return (
    <div style={{ padding: 24 }}>
      <h1>Organisation Units</h1>
      {loading ? (
        <CircularProgress />
      ) : error ? (
        <p style={{ color: 'red' }}>{error}</p>
      ) : (
        <>
          <Button variant="contained" onClick={handleOpen} sx={{ mb: 2 }}>
            New Organisation Unit
          </Button>
          {units.length === 0 ? (
            <p>No organisation units available.</p>
          ) : (
            <TreeView>
              {renderTree(treeData)}
            </TreeView>
          )}
        </>
      )}
      <Dialog open={openDialog} onClose={handleClose} fullWidth>
        <DialogTitle>Add Organisation Unit</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Name"
            fullWidth
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
          <TextField
            select
            margin="dense"
            label="Parent"
            fullWidth
            value={parentId}
            onChange={(e) => setParentId(e.target.value === '' ? '' : Number(e.target.value))}
          >
            <MenuItem value="">No parent</MenuItem>
            {units.map((u) => (
              <MenuItem key={u.id} value={u.id}>{u.name}</MenuItem>
            ))}
          </TextField>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancel</Button>
          <Button onClick={handleCreate} disabled={!newName}>Create</Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}