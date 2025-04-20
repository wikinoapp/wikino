# typed: true
# frozen_string_literal: true

module ModelConcerns
  module Viewable
    extend T::Sig

    sig { overridable.returns(String) }
    attr_accessor :serialized_locale

    sig { overridable.returns(String) }
    attr_accessor :time_zone

    sig { overridable.returns(T::Boolean) }
    def signed_in?
      false
    end

    sig { overridable.returns(Locale) }
    def locale
      Locale.deserialize(serialized_locale)
    end
  end
end
