# typed: strict
# frozen_string_literal: true

class ApplicationService
  extend T::Sig

  class RecordNotUniqueError < StandardError
    extend T::Sig

    sig { returns(Symbol) }
    attr_reader :attribute

    sig { params(message: String, attribute: Symbol).void }
    def initialize(message:, attribute: :base)
      super(message)
      @attribute = attribute
    end
  end

  sig { params(block: T.proc.void).void }
  private def with_transaction(&block)
    ApplicationRecord.transaction(&block)
  rescue ActiveRecord::RecordNotUnique => e
    message = I18n.t("services.errors.messages.uniqueness")

    # PostgreSQLの場合のエラーメッセージ例:
    # "PG::UniqueViolation: ERROR: duplicate key value violates unique constraint "index_pages_on_slug""
    case e.message
    when /index_space_members_on_space_id_and_user_id/
      raise RecordNotUniqueError.new(message:, attribute: :identifier)
    else
      # 予期しない一意性制約違反はシステムエラーとして扱う
      raise
    end
  end
end
