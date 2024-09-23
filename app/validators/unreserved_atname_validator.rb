# typed: strict
# frozen_string_literal: true

class UnreservedAtnameValidator < ActiveModel::EachValidator
  extend T::Sig

  sig { params(record: T.any(AccountForm, User), attribute: Symbol, atname: String).void }
  def validate_each(record, attribute, atname)
    return if atname.blank?

    if reserved?(atname)
      record.errors.add(attribute, :reserved)
    end
  end

  sig { params(atname: String).returns(T::Boolean) }
  private def reserved?(atname)
    Nonoto.config.reserved_atnames.include?(atname)
  end
end
