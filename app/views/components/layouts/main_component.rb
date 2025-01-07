# typed: strict
# frozen_string_literal: true

module Layouts
  class MainComponent < ApplicationComponent
    use_helpers :signed_in?
  end
end
