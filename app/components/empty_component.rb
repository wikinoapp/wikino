# frozen_string_literal: true

class EmptyComponent < ApplicationComponent
  def initialize(message:)
    @message = message
  end
end
