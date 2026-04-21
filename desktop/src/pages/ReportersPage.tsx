import React, { useEffect, useState } from 'react';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  MenuItem,
  CircularProgress,
  IconButton,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';

interface Reporter {
  id: number;
  uid: string;
  rapidProUuid: string;
  phoneNumber: string;
  displayName: string;
  orgUnitId: number;
  isActive: boolean;
}

interface OrgUnit {
  id: number;
  name: string;
}

export default function ReportersPage() {
  const [reporters, setReporters] = useState<Reporter[]>([]);
  const [orgUnits, setOrgUnits] = useState<OrgUnit[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [rapidProUuid, setRapidProUuid] = useState('');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [orgUnitId, setOrgUnitId] = useState<number | ''>('');

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const res = await fetch('/api/v1/sukumad/reporters?page=0&pageSize=100');
        if (!res.ok) throw new Error(`reporters HTTP ${res.status}`);
        const json = await res.json();
        const items = json.items || json;
        setReporters(items);
        const resOu = await fetch('/api/v1/sukumad/org_units');
        if (!resOu.ok) throw new Error(`org units HTTP ${resOu.status}`);
        const ouJson = await resOu.json();
        setOrgUnits(ouJson.items || ouJson);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  const handleOpen = () => {
    setRapidProUuid('');
    setPhoneNumber('');
    setDisplayName('');
    setOrgUnitId('');
    setOpenDialog(true);
  };
  const handleClose = () => setOpenDialog(false);
  const handleCreate = async () => {
    try {
      const payload = {
        rapidProUuid,
        phoneNumber,
        displayName,
        orgUnitId: orgUnitId === '' ? null : orgUnitId,
      };
      const res = await fetch('/api/v1/sukumad/reporters', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const created = await res.json();
      setReporters((prev) => [...prev, created]);
      handleClose();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Delete this reporter?')) return;
    try {
      const res = await fetch(`/api/v1/sukumad/reporters/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setReporters((prev) => prev.filter((r) => r.id !== id));
    } catch (err: any) {
      alert(err.message);
    }
  };

  const columns: GridColDef[] = [
    { field: 'displayName', headerName: 'Name', flex: 1 },
    { field: 'phoneNumber', headerName: 'Phone', flex: 1 },
    {
      field: 'orgUnitName',
      headerName: 'Organisation Unit',
      flex: 1,
      valueGetter: (params) => {
        const org = orgUnits.find((o) => o.id === params.row.orgUnitId);
        return org ? org.name : '';
      },
    },
    { field: 'isActive', headerName: 'Active', width: 100 },
    {
      field: 'actions',
      headerName: '',
      width: 80,
      renderCell: (params) => (
        <IconButton onClick={() => handleDelete(params.row.id)}>
          <DeleteIcon />
        </IconButton>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <h1>Reporters</h1>
      {loading ? (
        <CircularProgress />
      ) : error ? (
        <p style={{ color: 'red' }}>{error}</p>
      ) : (
        <>
          <Button variant="contained" onClick={handleOpen} sx={{ mb: 2 }}>
            New Reporter
          </Button>
          <div style={{ height: 400, width: '100%' }}>
            <DataGrid rows={reporters} columns={columns} autoHeight disableSelectionOnClick />
          </div>
        </>
      )}
      <Dialog open={openDialog} onClose={handleClose} fullWidth>
        <DialogTitle>Add Reporter</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="RapidPro UUID"
            fullWidth
            value={rapidProUuid}
            onChange={(e) => setRapidProUuid(e.target.value)}
          />
          <TextField
            margin="dense"
            label="Phone Number"
            fullWidth
            value={phoneNumber}
            onChange={(e) => setPhoneNumber(e.target.value)}
          />
          <TextField
            margin="dense"
            label="Display Name"
            fullWidth
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
          <TextField
            select
            margin="dense"
            label="Organisation Unit"
            fullWidth
            value={orgUnitId}
            onChange={(e) => setOrgUnitId(e.target.value === '' ? '' : Number(e.target.value))}
          >
            <MenuItem value="">Select org unit</MenuItem>
            {orgUnits.map((u) => (
              <MenuItem key={u.id} value={u.id}>{u.name}</MenuItem>
            ))}
          </TextField>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancel</Button>
          <Button
            onClick={handleCreate}
            disabled={!phoneNumber || orgUnitId === ''}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}
