# typed: strict
# frozen_string_literal: true

require "rotp"
require "rqrcode"

module TwoFactorAuthService
  class Setup < ApplicationService
    sig { params(user: User).returns(SetupResult) }
    def call(user:)
      # Generate a new TOTP secret
      secret = ROTP::Base32.random

      # Create TOTP instance
      totp = ROTP::TOTP.new(secret, issuer: "Wikino")

      # Generate provisioning URI
      provisioning_uri = totp.provisioning_uri(user.email)

      # Generate QR code as SVG
      qr_code = generate_qr_code_svg(provisioning_uri)

      # Find or create UserTwoFactorAuth record
      auth_record = UserTwoFactorAuthRecord.find_or_initialize_by(user_id: user.database_id)

      # Update the record with new secret (but don't enable it yet)
      auth_record.update!(
        secret: secret,
        enabled: false,
        recovery_codes: []
      )

      SetupResult.new(
        success: true,
        secret: secret,
        provisioning_uri: provisioning_uri,
        qr_code: qr_code
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

      # Generate SVG with reasonable size
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
