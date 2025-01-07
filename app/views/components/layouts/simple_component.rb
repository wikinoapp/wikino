# typed: strict
# frozen_string_literal: true

module Layouts
  class SimpleComponent < ApplicationComponent
    use_helpers :signed_in?
  end
end
