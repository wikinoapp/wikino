# typed: strict
# frozen_string_literal: true

module SoftDeletable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    scope :deleted, -> { where.not(deleted_at: nil) }
    scope :only_kept, -> { where(deleted_at: nil) }

    sig { returns(T::Boolean) }
    def not_deleted?
      deleted_at.nil?
    end

    sig { returns(T::Boolean) }
    def deleted?
      !not_deleted?
    end
  end
end
