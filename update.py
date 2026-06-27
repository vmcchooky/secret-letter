with open('/etc/caddy/Caddyfile', 'r') as f:
    lines = f.read().splitlines()

# find where moon.quorix.io.vn block ends (it ends at the first '}' after line 73)
idx = -1
for i, line in enumerate(lines):
    if line.strip() == '}' and "moon.quorix.io.vn" in '\n'.join(lines[:i]):
        idx = i

# wait, just keep lines until line 74
clean_lines = lines[:74]

clean_lines.append('')
clean_lines.append('www.secret.quorix.io.vn {')
clean_lines.append('    redir https://secret.quorix.io.vn{uri}')
clean_lines.append('}')

with open('/tmp/Caddyfile', 'w', newline='\n') as f:
    f.write('\n'.join(clean_lines) + '\n')
