# frozen_string_literal: true

module SoftDeletable
  extend ActiveSupport::Concern

  included do
    scope :deleted, -> { where.not(deleted_at: nil) }
    scope :only_kept, -> { where(deleted_at: nil) }

    def not_deleted?
      deleted_at.nil?
    end

    def deleted?
      !not_deleted?
    end
  end
end
