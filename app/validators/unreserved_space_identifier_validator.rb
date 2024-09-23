# typed: strict
# frozen_string_literal: true

class UnreservedSpaceIdentifierValidator < ActiveModel::EachValidator
  extend T::Sig

  sig { params(record: AccountForm, attribute: Symbol, identifier: String).void }
  def validate_each(record, attribute, identifier)
    return if identifier.blank?

    if reserved?(identifier)
      record.errors.add(attribute, :reserved)
    end
  end

  sig { params(identifier: String).returns(T::Boolean) }
  private def reserved?(identifier)
    Nonoto.config.reserved_space_identifiers.include?(identifier)
  end
end
