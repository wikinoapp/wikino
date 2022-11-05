# typed: strict
# frozen_string_literal: true

class ApplicationComponent < ViewComponent::Base
  extend T::Sig

  delegate :user_signed_in?, to: :helpers
end
