# typed: strict
# frozen_string_literal: true

module AccountService
  class Destroy < ApplicationService
    sig { params(user_record_id: T::Wikino::DatabaseId).void }
    def call(user_record_id:)
      user_record = UserRecord.find(user_record_id)

      user_record.user_session_records.destroy_all
      user_record.user_password_record&.destroy!

      # reload しないと user_password_record が存在するものとして例外が発生する
      user_record.reload.destroy!

      nil
    end
  end
end
