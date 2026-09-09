# セキュリティ

本番ではWebhook secretを平文保存せず、KMSやSecret Managerを利用してください。受信先URLのSSRF対策、HTTPS強制、送信先allowlist、本文サイズ制限を追加してから外部公開します。
