# typed: strict
# frozen_string_literal: true

module PasswordService
  class Update < ApplicationService
    sig { params(email: String, password: String).void }
    def call(email:, password:)
      user_record = UserRecord.find_by!(email:)

      user_record.user_password_record.not_nil!.update!(password:)

      nil
    end
  end
end
