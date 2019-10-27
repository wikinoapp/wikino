# frozen_string_literal: true

module SoftDeletable
  extend ActiveSupport::Concern

  included do
    scope :without_deleted, -> { where(deleted_at: nil) }

    def soft_delete
      touch :deleted_at
    end
  end
end
