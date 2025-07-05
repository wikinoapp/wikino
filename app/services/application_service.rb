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

  sig do
    type_parameters(:T)
      .params(block: T.proc.returns(T.type_parameter(:T)))
      .returns(T.type_parameter(:T))
  end
  private def with_transaction(&block)
    ApplicationRecord.transaction(&block)
  rescue ActiveRecord::RecordNotUnique => e
    message = I18n.t("services.errors.messages.uniqueness")

    # PostgreSQLの場合のエラーメッセージ例:
    # "PG::UniqueViolation: ERROR: duplicate key value violates unique constraint "index_pages_on_slug""
    case e.message
    when /index_spaces_on_identifier/
      raise RecordNotUniqueError.new(message:, attribute: :identifier)
    else
      # 予期しない一意性制約違反はシステムエラーとして扱う
      raise
    end
  end
end
