# typed: strict
# frozen_string_literal: true

class Markup
  extend T::Sig

  sig { params(text: String).void }
  def initialize(text:)
    @text = text
  end

  sig { returns(String) }
  def render_html
    Commonmarker.to_html(text, options: {
      render: {hardbreaks: false}
    })
  end

  sig { returns(String) }
  attr_reader :text
  private :text
end
