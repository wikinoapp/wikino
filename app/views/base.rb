# typed: strict
# frozen_string_literal: true

module Views
  class Base < ViewComponent::Base
    extend T::Sig

    VC = Views::Components
  end
end
