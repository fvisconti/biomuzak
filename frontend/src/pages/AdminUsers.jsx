import React, {useEffect, useState} from 'react';
import { Box, Button, Table, Thead, Tbody, Tr, Th, Td, TableContainer, Heading, Spinner } from '@chakra-ui/react';
import axios from 'axios';
import { useAuth } from '../context/AuthContext';

const AdminUsers = () => {
  const { user } = useAuth();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(null);

  useEffect(() => {
    const fetchUsers = async () => {
      try {
        const res = await axios.get('/api/admin/users');
        setUsers(res.data || []);
      } catch (err) {
        console.error('Failed fetching users', err);
      } finally {
        setLoading(false);
      }
    };
    fetchUsers();
  }, []);

  const handleDelete = async (id) => {
    if (!window.confirm('Delete user #' + id + '?')) return;
    setDeleting(id);
    try {
      await axios.delete(`/api/admin/users/${id}`);
      setUsers(prev => prev.filter(u => u.id !== id));
    } catch (err) {
      console.error('Failed to delete user', err);
      alert('Failed to delete user');
    } finally {
      setDeleting(null);
    }
  };

  if (!user || !user.is_admin) return <Box p={6}>Access denied</Box>;

  return (
    <Box p={6}>
      <Heading size="md" mb={4}>Admin — Users</Heading>
      {loading ? <Spinner /> : (
        <TableContainer>
          <Table variant="simple">
            <Thead>
              <Tr><Th>ID</Th><Th>Username</Th><Th>Email</Th><Th>Admin</Th><Th>Actions</Th></Tr>
            </Thead>
            <Tbody>
              {users.map(u => (
                <Tr key={u.id}>
                  <Td>{u.id}</Td>
                  <Td>{u.username}</Td>
                  <Td>{u.email}</Td>
                  <Td>{u.is_admin ? 'yes' : 'no'}</Td>
                  <Td>
                    <Button colorScheme="red" size="sm" onClick={() => handleDelete(u.id)} isLoading={deleting===u.id}>Delete</Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
};

export default AdminUsers;
