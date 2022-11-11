# typed: strict
# frozen_string_literal: true

module ApplicationHelper
  extend T::Sig

  sig { returns(String) }
  def nonoto_display_meta_tags
    display_meta_tags(
      reverse: true,
      site: "Nonoto",
      separator: " |",
      description: "A note taking app.",
    )
  end
end
