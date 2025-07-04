# typed: strict
# frozen_string_literal: true

require "rotp"
require "rqrcode"

module TwoFactorAuthService
  class Setup < ApplicationService
    sig { params(user: User).returns(SetupResult) }
    def call(user:)
      # 新しいTOTPシークレットを生成
      secret = ROTP::Base32.random

      # TOTPインスタンスを作成
      totp = ROTP::TOTP.new(secret, issuer: "Wikino")

      # プロビジョニングURIを生成
      provisioning_uri = totp.provisioning_uri(user.email)

      # QRコードをSVG形式で生成
      qr_code = generate_qr_code_svg(provisioning_uri)

      # UserTwoFactorAuthレコードを検索または作成
      auth_record = UserTwoFactorAuthRecord.find_or_initialize_by(user_id: user.database_id)

      # 新しいシークレットでレコードを更新 (まだ有効化はしない)
      auth_record.update!(
        secret:,
        enabled: false,
        recovery_codes: []
      )

      SetupResult.new(
        success: true,
        secret:,
        provisioning_uri:,
        qr_code:
      )
    rescue => e
      Rails.logger.error("Failed to setup 2FA for user #{user.database_id}: #{e.message}")
      SetupResult.new(
        success: false,
        secret: "",
        provisioning_uri: "",
        qr_code: nil
      )
    end

    class SetupResult < T::Struct
      const :success, T::Boolean
      const :secret, String, sensitivity: []
      const :provisioning_uri, String
      const :qr_code, T.nilable(String)
    end

    sig { params(data: String).returns(String) }
    private def generate_qr_code_svg(data)
      qrcode = RQRCode::QRCode.new(data)

      # 適切なサイズでSVGを生成
      qrcode.as_svg(
        offset: 0,
        color: "000",
        shape_rendering: "crispEdges",
        module_size: 4,
        standalone: true,
        svg_attributes: {
          width: 200,
          height: 200
        }
      )
    end
  end
end
