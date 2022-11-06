# typed: strict
# frozen_string_literal: true

class FlashComponent < ApplicationComponent
  sig { params(flash: ActionDispatch::Flash::FlashHash).void }
  def initialize(flash)
    @flash = flash
  end

  private

  sig { returns(T::Boolean) }
  def render?
    !@flash.empty?
  end

  sig { returns(T.nilable(Symbol)) }
  def flash_type
    @flash.keys.first&.to_sym
  end

  sig { returns(String) }
  def alert_bg_class
    case flash_type
    when :alert
      "alert-danger"
    when :notice
      "alert-success"
    else
      ""
    end
  end

  sig { returns(String) }
  def icon_class
    case flash_type
    when :alert
      "fa-exclamation-triangle"
    when :notice
      "fa-check-circle"
    else
      ""
    end
  end
end
