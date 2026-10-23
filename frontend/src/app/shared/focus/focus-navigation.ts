import {
  afterNextRender,
  Directive,
  ElementRef,
  inject,
  input,
  output,
  PLATFORM_ID,
} from '@angular/core';
import { isPlatformBrowser } from '@angular/common';

@Directive({
  selector: '[focusItem]',
})
export class FocusItem {
  readonly elementRef = inject(ElementRef);
}

@Directive({
  selector: '[focusGroup]',
  host: {
    '(keydown)': 'handleKeydown($event)',
  },
})
export class FocusGroup {
  private elementRef = inject(ElementRef);
  private platformId = inject(PLATFORM_ID);

  readonly autofocus = input<boolean>(false);
  readonly back = output<void>();

  constructor() {
    afterNextRender(() => {
      if (!isPlatformBrowser(this.platformId) || !this.autofocus()) {
        return;
      }
      const host = this.elementRef.nativeElement as HTMLElement;
      const first = host.querySelector('[focusItem]') as HTMLElement | null;
      first?.focus();
    });
  }

  handleKeydown(event: KeyboardEvent): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    const target = event.target as HTMLElement | null;
    const isTextInput =
      target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA');

    if (event.key === 'Escape' || event.key === 'Backspace') {
      if (!isTextInput) {
        event.preventDefault();
        this.back.emit();
      }
      return;
    }

    const arrowDirections: Record<string, 'up' | 'down' | 'left' | 'right'> = {
      ArrowUp: 'up',
      ArrowDown: 'down',
      ArrowLeft: 'left',
      ArrowRight: 'right',
    };

    const direction = arrowDirections[event.key];
    if (!direction) {
      return;
    }

    const host = this.elementRef.nativeElement as HTMLElement;
    const elements = Array.from(host.querySelectorAll('[focusItem]')) as HTMLElement[];
    const items = elements.filter(
      (item) => !item.hasAttribute('disabled') && item.offsetParent !== null
    );

    if (items.length === 0) {
      return;
    }

    const activeElement = document.activeElement as HTMLElement | null;
    if (!activeElement || !items.includes(activeElement)) {
      items[0].focus();
      event.preventDefault();
      return;
    }

    const currentRect = activeElement.getBoundingClientRect();
    const currentCx = currentRect.left + currentRect.width / 2;
    const currentCy = currentRect.top + currentRect.height / 2;

    let bestCandidate: HTMLElement | null = null;
    let shortestDistance = Number.POSITIVE_INFINITY;

    for (const item of items) {
      if (item === activeElement) {
        continue;
      }

      const rect = item.getBoundingClientRect();
      const cx = rect.left + rect.width / 2;
      const cy = rect.top + rect.height / 2;

      let isValidDirection = false;
      let primaryDiff = 0;
      let secondaryDiff = 0;

      switch (direction) {
        case 'up':
          isValidDirection = cy < currentCy - 2;
          primaryDiff = currentCy - cy;
          secondaryDiff = Math.abs(cx - currentCx);
          break;
        case 'down':
          isValidDirection = cy > currentCy + 2;
          primaryDiff = cy - currentCy;
          secondaryDiff = Math.abs(cx - currentCx);
          break;
        case 'left':
          isValidDirection = cx < currentCx - 2;
          primaryDiff = currentCx - cx;
          secondaryDiff = Math.abs(cy - currentCy);
          break;
        case 'right':
          isValidDirection = cx > currentCx + 2;
          primaryDiff = cx - currentCx;
          secondaryDiff = Math.abs(cy - currentCy);
          break;
      }

      if (!isValidDirection) {
        continue;
      }

      const distance = primaryDiff * primaryDiff + secondaryDiff * secondaryDiff * 2;
      if (distance < shortestDistance) {
        shortestDistance = distance;
        bestCandidate = item;
      }
    }

    if (bestCandidate) {
      bestCandidate.focus();
      event.preventDefault();
    }
  }
}
