# typed: strict
# frozen_string_literal: true

module AccountService
  class SoftDestroy < ApplicationService
    sig { params(user_record: UserRecord).void }
    def call(user_record:)
      user_record.discard!

      DestroyAccountJob.perform_later(user_record_id: user_record.id)

      nil
    end
  end
end
