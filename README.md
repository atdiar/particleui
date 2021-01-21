# ParticleUI

ParticleUI defines a set of objects and functions used for the creation of platform-independent,
easily composable, component based ui toolkits.

It defines a gui as a set of elements which hold properties that are either rendered
on screen or used to implement the logic of screen rendering.

These properties can be bound to each other (reactivity, mutation observing, it's all the same, nothing too fancy).
Typically an element can watch its children constituents for property change.
The children elements are encapsulated so that they expose an interface to the outer parent for communication via getter and setters.

Behaviors can be specified via the adjunction of Event Handlers to an Element.
These event handlers are modeled after the Basic DOM event.

(tbc ...)
