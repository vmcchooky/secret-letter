# Failover Runbook Cho One Time Link

## 1. Muc tieu

Runbook nay mo ta cach failover khi:

- `primary` la VPS nha cung cap o Viet Nam
- `standby` la Oracle Cloud VPS
- frontend van chay tren Vercel
- API domain la `api.secret.quorix.io.vn`

Runbook nay uu tien:

- de thuc hien
- de hoc
- de dung khi gap su co that

Khong uu tien:

- tu dong hoa som
- active-active

## 2. Gia dinh hien tai

Runbook nay gia dinh rang:

- frontend dang goi API qua `https://api.secret.quorix.io.vn`
- ca primary va standby deu co the chay duoc binary Go
- standby Oracle Cloud da duoc provision san o muc co ban
- ban co quyen sua DNS tai PA Viet Nam
- Redis dang chay local tren tung node

## 3. Dieu quan trong can hieu truoc

Failover trong mo hinh nay la:

- chuyen traffic tu primary sang standby

Chu khong phai:

- giu nguyen moi secret dang ton tai ma khong mat gi

### Ly do

Neu primary chet dot ngot, nhung secret dang nam trong Redis primary co the mat.

Day la danh doi hop ly cho MVP portfolio, vi:

- secret la du lieu ngan han
- one-time-link uu tien khong lo secret hon la phai backup tat ca secret
- he thong can de hieu, de van hanh

## 4. Khi nao can kich hoat failover

Failover nen duoc bat dau khi co it nhat mot trong cac dau hieu sau:

- `api.secret.quorix.io.vn/healthz` khong phan hoi
- SSH vao primary khong duoc
- Redis tren primary chet va khong the khoi phuc nhanh
- VPS primary gap su co mang, disk, hoac reboot loi keo dai
- thoi gian downtime vuot qua nguong ban chap nhan, vi du `10-15 phut`

## 5. Muc tieu sau failover

Sau khi failover xong, he thong can dat toi thieu:

- `api.secret.quorix.io.vn` tro toi Oracle Cloud VPS
- frontend goi duoc API
- create secret hoat dong
- reveal secret hoat dong voi secret moi tao sau failover
- health endpoint xanh

Khong bat buoc:

- phuc hoi tat ca secret cu dang ton tai tren primary truoc luc loi

## 6. Chuan bi truoc khi co su co

Nhung thu nay nen lam san:

- binary Go moi nhat co san tren standby
- file cau hinh app co san tren standby
- Caddyfile hoac Nginx config co san tren standby
- Redis da cai san tren standby
- systemd unit da duoc viet san tren standby
- domain `api.secret.quorix.io.vn` dung TTL thap, vi du `60-300 giay`
- ban biet cach sua DNS tai PA Viet Nam
- ban da test it nhat 1 lan failover gia lap

## 7. Kiem tra ban dau khi co canh bao

### 7.1 Xac nhan primary dang loi that

1. Ping domain API.
2. Goi `GET /healthz`.
3. Thu SSH vao primary.
4. Neu vao duoc, kiem tra app log va Redis.

### 7.2 Danh gia co can failover khong

Neu loi co the sua trong vai phut, ban co the sua tai cho.

Neu khong chac sua nhanh duoc, failover som thuong tot hon de giam downtime.

## 8. Quy trinh failover bang tay

## Buoc 1: Dong bang thay doi tren primary

Neu primary van con song nhung dang loi mot phan:

- dung deploy moi
- khong restart lien tuc khong kiem soat
- ghi lai trang thai hien tai

### Ly do

Khi su co xay ra, dieu nguy hiem la vua chua vua doi cau hinh, lam mat dau vet va kho debug.

## Buoc 2: Dang nhap vao Oracle standby

1. SSH vao Oracle node.
2. Kiem tra disk, RAM, va network.
3. Xac nhan binary, env, reverse proxy, Redis da co san.

Lenh kiem tra goi y:

```bash
hostname
date
df -h
free -m
systemctl status redis
systemctl status one-time-link
```

## Buoc 3: Khoi dong service tren standby

Neu service chua chay:

```bash
sudo systemctl start redis
sudo systemctl start one-time-link
sudo systemctl start caddy
```

Neu dang dung Nginx thay vi Caddy, doi service cho phu hop.

## Buoc 4: Kiem tra app tren standby

Test local tren standby:

```bash
curl -i http://127.0.0.1:8080/healthz
```

Neu qua reverse proxy:

```bash
curl -i https://api.secret.quorix.io.vn/healthz
```

Neu DNS chua tro ve standby thi co the test bang IP va Host header:

```bash
curl -k -H "Host: api.secret.quorix.io.vn" https://<ORACLE_PUBLIC_IP>/healthz
```

## Buoc 5: Chuyen DNS sang Oracle node

Tai PA Viet Nam:

1. Tim record `A` cua `api.secret.quorix.io.vn`
2. Doi IP tu primary sang IP cua Oracle standby
3. Luu thay doi

### Luu y

- neu TTL da thap san, viec chuyen se nhanh hon
- mot so may khach van cache DNS trong vai phut

## Buoc 6: Xac nhan traffic da toi standby

Sau khi cap nhat DNS:

1. `nslookup api.secret.quorix.io.vn`
2. `dig api.secret.quorix.io.vn`
3. `curl https://api.secret.quorix.io.vn/healthz`

Neu co log request tren standby, kiem tra xem request da vao Oracle node chua.

## Buoc 7: Test nghiep vu toi thieu

Sau khi DNS chuyen:

1. Tao mot secret moi.
2. Mo link moi.
3. Bam `Reveal`.
4. Thu bam lan hai va xac nhan `already used`.

### Vi sao can test secret moi

Secret cu tao truoc su co co the da mat cung Redis primary, nen test secret moi se do dung he thong sau failover chinh xac hon.

## Buoc 8: Ghi nhan su co

Can ghi it nhat:

- thoi diem bat dau su co
- ly do kich hoat failover
- thoi diem chuyen DNS
- thoi diem app phuc vu lai thanh cong
- symptom tren primary
- cach khoi phuc sau cung

## 9. Neu primary phuc hoi lai thi lam gi

Khi primary song lai, khong can switch nguoc ngay.

Nen lam theo thu tu:

1. Kiem tra nguyen nhan su co.
2. Dam bao primary thuc su on dinh.
3. Dong bo lai binary va config neu can.
4. Test local tren primary.
5. Chon thoi diem it traffic de failback.

### Ly do

Failback voi he thong chua on dinh co the tao downtime lan hai.

## 10. Quy trinh failback de xuat

## Buoc 1

Kiem tra primary da san sang:

```bash
curl -i http://127.0.0.1:8080/healthz
```

## Buoc 2

Test nghiep vu tren primary bang domain tam hoac bang IP + Host header.

## Buoc 3

Doi DNS `api.secret.quorix.io.vn` tro lai primary.

## Buoc 4

Kiem tra:

- DNS resolve dung
- health endpoint xanh
- create va reveal flow thanh cong

## Buoc 5

Giu standby Oracle o trang thai san sang cho lan sau.

## 11. Dieu gi co the bi mat khi failover

Nhung thu co the mat:

- secret chua reveal dang nam trong Redis primary
- reveal sessions dang ton tai tren primary

Nhung thu khong nen mat:

- source code
- config
- script deploy
- domain
- frontend tren Vercel

### Cach giam thiet hai

- chap nhan secret runtime la ephemeral
- luon giu source code va config trong Git
- giu standby da duoc provision san

## 12. Kich ban test failover hang thang

Ban nen tap failover it nhat moi thang 1 lan:

1. Ghi lai DNS hien tai.
2. Dung primary co chu y.
3. Khoi dong standby.
4. Chuyen DNS.
5. Test full flow.
6. Chuyen lai.
7. Rut kinh nghiem.

### Ly do

Runbook chi co gia tri khi ban da tap no.

## 13. Tieu chi danh gia failover thanh cong

Mot lan failover duoc xem la thanh cong khi:

- downtime nam trong gioi han ban chap nhan
- DNS chuyen dung
- health endpoint xanh
- create flow dung
- reveal flow dung
- khong co secret plaintext bi lo

## 14. Toi uu sau nay, nhung chua can ngay

Sau nay ban co the nang cap:

- dong bo artifact tu dong sang standby
- backup config tu dong
- canh bao uptime tu dong
- failover DNS nhanh hon
- readiness va smoke test tu dong

Nhung hien tai chua can:

- Kubernetes
- service mesh
- active-active multi-region
- Redis replication phuc tap xuyen provider

## 15. Ket luan cuoi

Muc tieu cua failover runbook cho du an nay khong phai la dat do san sang enterprise.

Muc tieu la:

- giam downtime
- co cach ung pho ro rang
- hoc duoc tu duy van hanh
- co cau chuyen deployment dep cho portfolio

Neu ban lam tot runbook nay, recruiter se thay duoc rang ban khong chi biet viet code, ma con biet nghi den van hanh, trade-off, va kha nang khoi phuc he thong.
