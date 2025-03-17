# typed: strict
# frozen_string_literal: true

class UnreservedAtnameValidator < ActiveModel::EachValidator
  extend T::Sig

  sig { params(record: T.any(AccountForm, User), attribute: Symbol, atname: T.nilable(String)).void }
  def validate_each(record, attribute, atname)
    return if atname.blank?

    if reserved?(atname.not_nil!)
      record.errors.add(attribute, :reserved)
    end
  end

  sig { params(atname: String).returns(T::Boolean) }
  private def reserved?(atname)
    Wikino.config.reserved_atnames.include?(atname)
  end
end
